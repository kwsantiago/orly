package relay

import (
	"errors"
	"net/http"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/protocol/socketapi"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/normalize"
	"strings"
)

func (s *Server) AddEvent(
	c context.T, rl relay.I, ev *event.E,
	hr *http.Request, origin string,
	authedPubkey []byte,
) (accepted bool, message []byte) {

	if ev == nil {
		return false, normalize.Invalid.F("empty event")
	}
	if ev.Kind.IsEphemeral() {
	} else {
		if saveErr := s.Publish(c, ev); saveErr != nil {
			if errors.Is(saveErr, store.ErrDupEvent) {
				return false, []byte(saveErr.Error())
			}
			errmsg := saveErr.Error()
			if socketapi.NIP20prefixmatcher.MatchString(errmsg) {
				if strings.Contains(errmsg, "tombstone") {
					return false, normalize.Error.F("event was deleted, not storing it again")
				}
				if strings.HasPrefix(errmsg, string(normalize.Blocked)) {
					return false, []byte(errmsg)
				}
				return false, []byte(errmsg)
			} else {
				return false, []byte(errmsg)
			}
		}
	}
	// notify subscribers
	s.listeners.Deliver(ev)
	accepted = true
	return
}
