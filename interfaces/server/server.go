package server

import (
	"net/http"
	"orly.dev/app/relay/publish"
	"orly.dev/encoders/event"
	"orly.dev/encoders/filters"
	"orly.dev/interfaces/relay"
	"orly.dev/interfaces/store"
	"orly.dev/utils/context"
)

type I interface {
	AcceptEvent(
		c context.T, ev *event.E, hr *http.Request, authedPubkey []byte,
		remote string,
	) (accept bool, notice string, afterSave func())
	AcceptReq(
		c context.T, hr *http.Request, f *filters.T,
		authedPubkey []byte, remote string,
	) (allowed *filters.T, accept bool, modified bool)
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
	PublicReadable() bool
	ServiceURL(req *http.Request) (s string)
}
