package server

import (
	"net/http"
	"orly.dev/encoders/event"
	"orly.dev/interfaces/store"
	"orly.dev/utils/context"
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
