package server

import (
	"net/http"
	"orly.dev/context"
	"orly.dev/event"
	"orly.dev/interfaces/store"
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
