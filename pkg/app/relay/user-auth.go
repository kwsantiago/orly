package relay

import (
	"bytes"
	"net/http"
	"orly.dev/pkg/protocol/httpauth"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"time"
)

func (s *Server) UserAuth(
	r *http.Request, remote string,
	tolerance ...time.Duration,
) (authed bool, pubkey []byte) {
	var valid bool
	var err error
	var tolerate time.Duration
	if len(tolerance) > 0 {
		tolerate = tolerance[0]
	}
	if valid, pubkey, err = httpauth.CheckAuth(r, tolerate); chk.E(err) {
		return
	}
	if !valid {
		log.E.F(
			"invalid auth %s from %s",
			r.Header.Get("Authorization"), remote,
		)
		return
	}
	for _, pk := range append(s.ownersFollowed, s.followedFollows...) {
		if bytes.Equal(pk, pubkey) {
			authed = true
			return
		}
	}
	return
}
