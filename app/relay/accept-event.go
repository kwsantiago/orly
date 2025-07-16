package relay

import (
	"net/http"
	"orly.dev/encoders/event"
	"orly.dev/utils/context"
)

func (s *Server) AcceptEvent(
	c context.T, ev *event.E, hr *http.Request, authedPubkey []byte,
	remote string,
) (accept bool, notice string, afterSave func()) {
	// if auth is required and the user is not authed, reject
	if s.AuthRequired() && len(authedPubkey) == 0 {
		return
	}
	accept = true
	return
}
