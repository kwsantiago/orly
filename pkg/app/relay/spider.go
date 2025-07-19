package relay

import (
	"bytes"
	"orly.dev/pkg/crypto/ec/bech32"
	"orly.dev/pkg/crypto/ec/schnorr"
	"orly.dev/pkg/encoders/bech32encoding"
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

func (s *Server) Spider() (err error) {
	log.I.S(s.C.Owners)
	for _, v := range s.C.Owners {
		var prf []byte
		var pk []byte
		var bits5 []byte
		if prf, bits5, err = bech32.DecodeNoLimit([]byte(v)); chk.D(err) {
			// try hex then
			if _, err = hex.DecBytes(pk, []byte(v)); chk.E(err) {
				log.W.F("owner key %s is neither bech32 npub nor hex", v)
				continue
			}
		} else {
			if !bytes.Equal(prf, bech32encoding.NpubHRP) {
				log.W.F("owner key %s is neither npub nor hex", v)
				continue
			}
			if pk, err = bech32.ConvertBits(bits5, 5, 8, false); chk.E(err) {
				continue
			}
		}
		// owners themselves are on the OwnersFollowed list as first level
		s.OwnersPubkeys = append(s.OwnersPubkeys, pk)
	}
	if len(s.OwnersPubkeys) == 0 {
		// there is no OwnersPubkeys, so there is nothing to do.
		return
	}
	log.I.S(s.OwnersPubkeys)
	owners := tag.New(s.OwnersPubkeys...)
	f := &filter.F{
		Kinds:   kinds.New(kind.FollowList),
		Authors: owners,
	}
	log.I.F("%s", f.Marshal(nil))
	// first search the local database
	var evs event.S
	if evs, err = s.Storage().QueryEvents(s.Ctx, f); chk.E(err) {
		// none were found, so we need to scan the spiders
		err = nil
	}
	if len(evs) < len(s.OwnersPubkeys) {
		// we need to search the spider seeds.
		for _, seed := range s.C.SpiderSeeds {
			var cli *ws.Client
			if cli, err = ws.RelayConnect(context.Bg(), seed); chk.E(err) {
				err = nil
				continue
			}
			var evss event.S
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
		// log.I.F("%s", ev.Marshal(nil))
		if _, _, err = s.Storage().SaveEvent(s.Ctx, ev); chk.E(err) {
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
				continue
			}
			pkMap[string(pk)] = struct{}{}
		}
	}
	for pk := range pkMap {
		s.OwnersFollowed = append(s.OwnersFollowed, []byte(pk))
	}
	own := "owner"
	if len(s.OwnersPubkeys) > 1 {
		own = "owners"
	}
	fol := "pubkey"
	if len(s.OwnersFollowed) > 1 {
		fol = "pubkeys"
	}
	log.T.F(
		"found %d %s with a total of %d followed %s",
		len(s.OwnersPubkeys), own, len(s.OwnersFollowed), fol,
	)
	// append the owners
	s.OwnersFollowed = append(s.OwnersFollowed, s.OwnersPubkeys...)
	return
}
