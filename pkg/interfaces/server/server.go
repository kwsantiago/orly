package server

import (
	"net/http"
	"orly.dev/pkg/app/config"
	"orly.dev/pkg/app/relay/publish"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/context"
	"time"
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
		c context.T, rl relay.I, ev *event.E, hr *http.Request, origin string,
		pubkey []byte,
	) (accepted bool, message []byte)
	AdminAuth(
		r *http.Request, remote string, tolerance ...time.Duration,
	) (authed bool, pubkey []byte)
	UserAuth(
		r *http.Request, remote string, tolerance ...time.Duration,
	) (authed bool, pubkey []byte, super bool)
	Context() context.T
	Publisher() *publish.S
	Publish(c context.T, evt *event.E) (err error)
	Relay() relay.I
	Shutdown()
	Storage() store.I
	AuthRequired() bool
	PublicReadable() bool
	ServiceURL(req *http.Request) (s string)
	OwnersPubkeys() (pks [][]byte)
	Config() *config.C
}
