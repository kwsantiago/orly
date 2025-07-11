package server

import (
	"net/http"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/event"
	"orly.dev/log"
)

func (s *S) AddEvent(
	c context.T, ev *event.E, hr *http.Request, remote string,
) (accepted bool, message []byte) {
	if !ev.Kind.IsEphemeral() {
		if _, _, err := s.Store.SaveEvent(c, ev); chk.E(err) {
			message = []byte(err.Error())
			return
		}
	} else {
		log.I.F("ephemeral event %s", ev.Serialize())
	}
	accepted = true
	return
}
