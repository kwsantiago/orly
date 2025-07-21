package relay

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/normalize"
)

var (
	NIP20prefixmatcher = regexp.MustCompile(`^\w+: `)
)

// AddEvent processes an incoming event, saves it if valid, and delivers it to
// subscribers.
//
// # Parameters
//
//   - c: context for request handling
//
//   - rl: relay interface
//
//   - ev: the event to be added
//
//   - hr: HTTP request related to the event (if any)
//
//   - origin: origin of the event (if any)
//
//   - authedPubkey: public key of the authenticated user (if any)
//
// # Return Values
//
//   - accepted: true if the event was successfully processed, false otherwise
//
//   - message: additional information or error message related to the
//     processing
//
// # Expected Behaviour:
//
// - Validates the incoming event.
//
// - Saves the event using the Publish method if it is not ephemeral.
//
// - Handles duplicate events by returning an appropriate error message.
//
// - Delivers the event to subscribers via the listeners' Deliver method.
//
// - Returns a boolean indicating whether the event was accepted and any
// relevant message.
func (s *Server) AddEvent(
	c context.T, rl relay.I, ev *event.E, hr *http.Request, origin string,
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
			if NIP20prefixmatcher.MatchString(errmsg) {
				if strings.Contains(errmsg, "tombstone") {
					return false, normalize.Error.F(
						"%s event was deleted, not storing it again",
						origin,
					)
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
