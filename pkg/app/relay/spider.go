package relay

import (
	"bytes"
	"orly.dev/pkg/crypto/ec/bech32"
	"orly.dev/pkg/encoders/bech32encoding"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
)

func (s *Server) Spider(noFetch ...bool) (err error) {
	var ownersPubkeys [][]byte
	for _, v := range s.C.Owners {
		var prf []byte
		var pk []byte
		var bits5 []byte
		if prf, bits5, err = bech32.DecodeNoLimit([]byte(v)); chk.D(err) {
			// try hex then
			if _, err = hex.DecBytes(pk, []byte(v)); chk.E(err) {
				log.W.F(
					"owner key %s is neither bech32 npub nor hex",
					v,
				)
				continue
			}
		} else {
			if !bytes.Equal(prf, bech32encoding.NpubHRP) {
				log.W.F(
					"owner key %s is neither bech32 npub nor hex",
					v,
				)
				continue
			}
			if pk, err = bech32.ConvertBits(bits5, 5, 8, false); chk.E(err) {
				continue
			}
		}
		// owners themselves are on the OwnersFollowed list as first level
		ownersPubkeys = append(ownersPubkeys, pk)
	}
	if len(ownersPubkeys) == 0 {
		// there is no OwnersPubkeys, so there is nothing to do.
		return
	}
	go func() {
		dontFetch := false
		if len(noFetch) > 0 && noFetch[0] {
			dontFetch = true
		}
		log.I.F("getting ownersFollowed")
		var ownersFollowed [][]byte
		if ownersFollowed, err = s.SpiderFetch(
			kinds.New(kind.FollowList), dontFetch, false, ownersPubkeys...,
		); chk.E(err) {
			return
		}
		// log.I.S(ownersFollowed)
		log.I.F("getting followedFollows")
		var followedFollows [][]byte
		if followedFollows, err = s.SpiderFetch(
			kinds.New(kind.FollowList), dontFetch, false, ownersFollowed...,
		); chk.E(err) {
			return
		}
		log.I.F("getting ownersMuted")
		var ownersMuted [][]byte
		if ownersMuted, err = s.SpiderFetch(
			kinds.New(kind.MuteList), dontFetch, false, ownersPubkeys...,
		); chk.E(err) {
			return
		}
		// remove the ownersFollowed and ownersMuted items from the followedFollows
		// list
		filteredFollows := make([][]byte, 0, len(followedFollows))
		for _, follow := range followedFollows {
			for _, owner := range ownersFollowed {
				if bytes.Equal(follow, owner) {
					break
				}
			}
			for _, owner := range ownersMuted {
				if bytes.Equal(follow, owner) {
					break
				}
			}
			filteredFollows = append(filteredFollows, follow)
		}
		followedFollows = filteredFollows
		own := "owner"
		if len(ownersPubkeys) > 1 {
			own = "owners"
		}
		fol := "pubkey"
		if len(ownersFollowed) > 1 {
			fol = "pubkeys"
		}
		folfol := "pubkey"
		if len(followedFollows) > 1 {
			folfol = "pubkeys"
		}
		mut := "pubkey"
		if len(ownersMuted) > 1 {
			mut = "pubkeys"
		}
		log.T.F(
			"found %d %s with a total of %d followed %s and %d followed's follows %s, and excluding %d owner muted %s",
			len(ownersPubkeys), own,
			len(ownersFollowed), fol,
			len(followedFollows), folfol,
			len(ownersMuted), mut,
		)
		// add the owners to the ownersFollowed
		ownersFollowed = append(ownersFollowed, ownersPubkeys...)
		s.SetOwnersPubkeys(ownersPubkeys)
		s.SetOwnersFollowed(ownersFollowed)
		s.SetFollowedFollows(followedFollows)
		s.SetOwnersMuted(ownersMuted)
		// lastly, update all followed users new events in the background
		if !dontFetch {
			go func() {
				everyone := append(ownersFollowed, followedFollows...)
				s.SpiderFetch(
					// kinds.New(
					// kind.ProfileMetadata, kind.RelayListMetadata,
					// kind.DMRelaysList,
					// ),
					nil,
					false, true, everyone...,
				)
			}()
		}
	}()
	return
}
