package server

import (
	"net/http"
	"not.realy.lol/context"
	"not.realy.lol/event"
)

func (s *S) AddEvent(
	c context.T, ev *event.E, hr *http.Request, remote string,
) (accepted bool, message []byte) {

	// TODO implement me

	panic("implement me")

}
