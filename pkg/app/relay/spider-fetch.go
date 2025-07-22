package relay

import (
	"orly.dev/pkg/crypto/ec/schnorr"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"sort"
	"sync"
)

func (s *Server) SpiderFetch(
	k *kinds.T, noFetch, noExtract bool, pubkeys ...[]byte,
) (pks [][]byte, err error) {
	// first search the local database
	pkList := tag.New(pubkeys...)
	f := &filter.F{
		Kinds:   k,
		Authors: pkList,
	}
	var evs event.S
	if evs, err = s.Storage().QueryEvents(s.Ctx, f); chk.E(err) {
		// none were found, so we need to scan the spiders
		err = nil
	}
	var kindsList string
	for i, kk := range k.K {
		if i > 0 {
			kindsList += ","
		}
		kindsList += kk.Name()
	}
	log.I.F("%d events found of type %s", len(evs), kindsList)
	// for _, ev := range evs {
	// 	o += fmt.Sprintf("%s\n\n", ev.Marshal(nil))
	// }
	// log.I.F("%s", o)
	if !noFetch {
		// we need to search the spider seeds.
		// Break up pubkeys into batches of 128
		for i := 0; i < len(pubkeys); i += 128 {
			end := i + 128
			if end > len(pubkeys) {
				end = len(pubkeys)
			}
			batchPubkeys := pubkeys[i:end]
			log.I.F(
				"processing batch %d to %d of %d for kind %s",
				i, end, len(pubkeys), kindsList,
			)
			batchPkList := tag.New(batchPubkeys...)
			lim := uint(batchPkList.Len())
			batchFilter := &filter.F{
				Kinds:   k,
				Authors: batchPkList,
				Limit:   &lim,
			}

			var mx sync.Mutex
			var wg sync.WaitGroup

			for _, seed := range s.C.SpiderSeeds {
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case <-s.Ctx.Done():
						return
					default:
					}
					var evss event.S
					var cli *ws.Client
					if cli, err = ws.RelayConnect(
						context.Bg(), seed,
					); chk.E(err) {
						err = nil
						return
					}
					if evss, err = cli.QuerySync(
						context.Bg(), batchFilter,
					); chk.E(err) {
						err = nil
						return
					}
					mx.Lock()
					// save the events to the database
					for _, ev := range evss {
						log.I.F("saving event:\n%s", ev.Marshal(nil))
						if _, _, err = s.Storage().SaveEvent(
							s.Ctx, ev,
						); chk.E(err) {
							err = nil
							continue
						}
					}
					for _, ev := range evss {
						evs = append(evs, ev)
					}
					mx.Unlock()
				}()
			}
			wg.Wait()
		}
	}
	// deduplicate and take the newest
	var tmp event.S
	evMap := make(map[string]event.S)
	for _, ev := range evs {
		evMap[ev.PubKeyString()] = append(evMap[ev.PubKeyString()], ev)
	}
	for _, evm := range evMap {
		if len(evm) < 1 {
			continue
		}
		if len(evm) > 1 {
			sort.Sort(evm)
		}
		tmp = append(tmp, evm[0])
	}
	evs = tmp
	// we have all we're going to get now, extract the p tags
	if noExtract {
		return
	}
	pkMap := make(map[string]struct{})
	for _, ev := range evs {
		t := ev.Tags.GetAll(tag.New("p"))
		for _, tt := range t.ToSliceOfTags() {
			pkh := tt.Value()
			if len(pkh) != 2*schnorr.PubKeyBytesLen {
				continue
			}
			pk := make([]byte, schnorr.PubKeyBytesLen)
			if _, err = hex.DecBytes(pk, pkh); err != nil {
				err = nil
				continue
			}
			pkMap[string(pk)] = struct{}{}
		}
	}
	for pk := range pkMap {
		pks = append(pks, []byte(pk))
	}
	log.I.F("found %d pks", len(pks))
	return
}
