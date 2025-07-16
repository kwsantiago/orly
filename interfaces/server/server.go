package server

import (
	"net/http"
	"orly.dev/app/realy/publish"
	"orly.dev/encoders/event"
	"orly.dev/interfaces/relay"
	"orly.dev/interfaces/store"
	"orly.dev/utils/context"
)

type S interface {
	AddEvent(
		c context.T, rl relay.I, ev *event.E, hr *http.Request,
		origin string, authedPubkey []byte,
	) (
		accepted bool,
		message []byte,
	)
	Context() context.T
	Disconnect()
	Publisher() *publish.S
	Publish(c context.T, evt *event.E) (err error)
	Relay() relay.I
	Shutdown()
	Storage() store.I
	AuthRequired() bool
	ServiceURL(req *http.Request) (s string)
}
