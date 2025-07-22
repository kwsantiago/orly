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
	"runtime/debug"
	"sync"
)

// IdPkTs is a map of event IDs to their id, pubkey, and timestamp
// This is used to reduce memory usage by storing only the essential information
// instead of the full events
type IdPkTs struct {
	Id        []byte
	Pubkey    []byte
	Timestamp int64
}

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
	// Map to store id, pubkey, and timestamp for each event
	idPkTsMap := make(map[string]*IdPkTs)
	if evs, err = s.Storage().QueryEvents(s.Ctx, f); chk.E(err) {
		// none were found, so we need to scan the spiders
		err = nil
	}
	// Extract id, pubkey, and timestamp from initial events
	for _, ev := range evs {
		idStr := ev.IdString()
		idPkTsMap[idStr] = &IdPkTs{
			Id:        ev.Id,
			Pubkey:    ev.Pubkey,
			Timestamp: ev.CreatedAtInt64(),
		}
	}
	var kindsList string
	for i, kk := range k.K {
		if i > 0 {
			kindsList += ","
		}
		kindsList += kk.Name()
	}
	log.I.F("%d events found of type %s", len(evs), kindsList)
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
					// save the events to the database and extract id, pubkey, and timestamp
					for i, ev := range evss {
						log.I.F("saving event:\n%s", ev.Marshal(nil))
						if _, _, err = s.Storage().SaveEvent(
							s.Ctx, ev,
						); chk.E(err) {
							err = nil
							continue
						}

						// Extract id, pubkey, and timestamp
						idStr := ev.IdString()
						mx.Lock()
						idPkTsMap[idStr] = &IdPkTs{
							Id:        ev.Id,
							Pubkey:    ev.Pubkey,
							Timestamp: ev.CreatedAtInt64(),
						}
						// Append the event to evs for further processing
						evs = append(evs, ev)
						mx.Unlock()

						// Nil the event in the slice to free memory
						evss[i] = nil
					}
					chk.E(s.Storage().Sync())
					debug.FreeOSMemory()
				}()
			}
			wg.Wait()
		}
	}
	// deduplicate and take the newest
	// We need to query the database for the events we need to extract p tags from
	// since we've niled the events in memory
	if noExtract {
		return
	}

	// Create a list of event IDs to query
	var eventIds [][]byte
	for _, idPkTs := range idPkTsMap {
		eventIds = append(eventIds, idPkTs.Id)
	}

	// Query the database for the events
	var eventsForExtraction event.S
	if len(eventIds) > 0 {
		// Create a filter for the event IDs
		idFilter := &filter.F{
			Ids: tag.New(eventIds...),
		}

		// Query the database
		if eventsForExtraction, err = s.Storage().QueryEvents(
			s.Ctx, idFilter,
		); chk.E(err) {
			err = nil
		}
	}

	// Extract the p tags
	pkMap := make(map[string]struct{})
	for _, ev := range eventsForExtraction {
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

		// Nil the event after extraction to free memory
		ev = nil
	}
	for pk := range pkMap {
		pks = append(pks, []byte(pk))
	}
	log.I.F("found %d pks", len(pks))
	return
}
