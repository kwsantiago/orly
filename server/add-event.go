package server

import (
	"net/http"
	"not.realy.lol/chk"
	"not.realy.lol/context"
	"not.realy.lol/event"
)

func (s *S) AddEvent(
	c context.T, ev *event.E, hr *http.Request, remote string,
) (accepted bool, message []byte) {
	if err := s.Store.SaveEvent(c, ev); chk.E(err) {
		message = []byte(err.Error())
		return
	}
	accepted = true
	return
}
