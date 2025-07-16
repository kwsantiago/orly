package relay

import (
	"net/http"
	"orly.dev/encoders/filters"
	"orly.dev/utils/context"
)

func (s *Server) AcceptReq(
	c context.T, hr *http.Request, f *filters.T,
	authedPubkey []byte, remote string,
) (allowed *filters.T, accept bool, modified bool) {
	// if auth is required, and not public readable, reject
	if s.AuthRequired() && len(authedPubkey) == 0 && !s.PublicReadable() {
		return
	}
	accept = true
	return
}
