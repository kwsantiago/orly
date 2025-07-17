// Package app implements the orly nostr relay.
package app

import (
	"net/http"
	"orly.dev/pkg/app/config"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/context"
	"sync"
)

type List map[string]struct{}

type Relay struct {
	sync.Mutex
	*config.C
	Store store.I
}

func (r *Relay) Name() string { return r.C.AppName }

func (r *Relay) Storage() store.I { return r.Store }

func (r *Relay) Init() (err error) {
	// for _, src := range r.C.Owners {
	//	if len(src) < 1 {
	//		continue
	//	}
	//	dst := make([]byte, len(src)/2)
	//	if _, err = hex.DecBytes(dst, []byte(src)); chk.E(err) {
	//		if dst, err = bech32encoding.NpubToBytes([]byte(src)); chk.E(err) {
	//			continue
	//		}
	//	}
	//	r.owners = append(r.owners, dst)
	// }
	// if len(r.owners) > 0 {
	//	log.F.C(func() string {
	//		ownerIds := make([]string, len(r.owners))
	//		for i, npub := range r.owners {
	//			ownerIds[i] = hex.Enc(npub)
	//		}
	//		owners := strings.Join(ownerIds, ",")
	//		return fmt.Sprintf("owners %s", owners)
	//	})
	//	r.ZeroLists()
	//	r.CheckOwnerLists(context.Bg())
	// }
	return nil
}

func (r *Relay) AcceptEvent(
	c context.T, evt *event.E, hr *http.Request,
	origin string, authedPubkey []byte,
) (accept bool, notice string, afterSave func()) {
	accept = true
	return
}

func (r *Relay) AcceptFilter(
	c context.T, hr *http.Request, f *filter.S,
	authedPubkey []byte,
) (allowed *filter.S, ok bool, modified bool) {
	allowed = f
	ok = true
	return
}

func (r *Relay) AcceptReq(
	c context.T, hr *http.Request, id []byte,
	ff *filters.T, authedPubkey []byte,
) (allowed *filters.T, ok bool, modified bool) {
	allowed = ff
	ok = true
	return
}
