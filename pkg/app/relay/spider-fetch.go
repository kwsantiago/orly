package relay

import (
	"orly.dev/pkg/crypto/ec/schnorr"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"sort"
)

func (s *Server) SpiderFetch(
	k *kind.T, pubkeys ...[]byte,
) (pks [][]byte, err error) {
	pkList := tag.New(pubkeys...)
	f := &filter.F{
		Kinds:   kinds.New(k),
		Authors: pkList,
	}
	// first search the local database
	var evs event.S
	if evs, err = s.Storage().QueryEvents(s.Ctx, f); chk.E(err) {
		// none were found, so we need to scan the spiders
		err = nil
	}
	if len(evs) < len(pubkeys) {
		// we need to search the spider seeds.
		for _, seed := range s.C.SpiderSeeds {
			var evss event.S
			var cli *ws.Client
			if cli, err = ws.RelayConnect(context.Bg(), seed); chk.E(err) {
				err = nil
				continue
			}
			if evss, err = cli.QuerySync(context.Bg(), f); chk.E(err) {
				err = nil
				continue
			}
			for _, ev := range evss {
				evs = append(evs, ev)
			}
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
	// save the events to the database
	for _, ev := range evs {
		if _, _, err = s.Storage().SaveEvent(s.Ctx, ev); chk.E(err) {
			err = nil
			continue
		}
	}
	// we have all we're going to get now
	pkMap := make(map[string]struct{})
	for _, ev := range evs {
		t := ev.Tags.GetAll(tag.New("p"))
		for _, tt := range t.ToSliceOfTags() {
			pkh := tt.Value()
			if len(pkh) != 2*schnorr.PubKeyBytesLen {
				continue
			}
			pk := make([]byte, schnorr.PubKeyBytesLen)
			if _, err = hex.DecBytes(pk, pkh); chk.E(err) {
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
