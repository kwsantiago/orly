package server

import (
	"net/http"
	"not.realy.lol/context"
	"not.realy.lol/event"
	"not.realy.lol/interfaces/store"
)

type I interface {
	Context() context.T
	HandleRelayInfo(
		w http.ResponseWriter, r *http.Request,
	)
	Storage() store.I
	AddEvent(
		c context.T, ev *event.E, hr *http.Request, remote string,
	) (accepted bool, message []byte)
}
