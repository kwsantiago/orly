package interfaces

import (
	"net/http"
	"orly.dev/app/realy/publish"
	"orly.dev/encoders/event"
	"orly.dev/interfaces/relay"
	"orly.dev/interfaces/store"
	"orly.dev/utils/context"
)

type Server interface {
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
	// Options() *options.T
	// AcceptEvent(
	//	c context.T, ev *event.E, hr *http.Request, origin string,
	//	authedPubkey []byte) (accept bool, notice string, afterSave func())
	// AdminAuth(r *http.Request,
	//	tolerance ...time.Duration) (authed bool, pubkey []byte)
	// AuthRequired() bool
	// Configuration() store.Configuration
	// Owners() [][]byte
	// PublicReadable() bool
	// SetConfiguration(*store.Configuration)
}
