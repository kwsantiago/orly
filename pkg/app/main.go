// Package app implements the orly nostr relay.
package app

import (
	"net/http"
	"sync"

	"orly.dev/pkg/app/config"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/context"
)

// List represents a set-like structure using a map with empty struct values.
type List map[string]struct{}

// Relay is a struct that represents a relay for Nostr events. It contains a
// configuration and a persistence layer for storing the events. The Relay
// type implements various methods to handle event acceptance, filtering,
// and storage.
type Relay struct {
	sync.Mutex
	*config.C
	Store store.I
}

// Name returns the name of the application represented by this relay.
//
// Return Values:
//
//   - string: the name of the application.
//
// Expected behaviour:
//
// This function simply returns the AppName field from the configuration.
func (r *Relay) Name() string { return r.C.AppName }

// Storage represents a persistence layer for Nostr events handled by a relay.
func (r *Relay) Storage() store.I { return r.Store }

// Init initializes and sets up the relay for Nostr events.
//
// Return Values:
//
//   - err: an error if any issues occurred during initialization.
//
// Expected behaviour:
//
// This function is responsible for setting up the relay, configuring it,
// and initializing the necessary components to handle Nostr events.
func (r *Relay) Init() (err error) {
	return nil
}

// AcceptEvent checks an event and determines whether the event should be
// accepted and if the client has the authority to submit it.
//
// Parameters:
//
//   - c - a context.T for signalling if the task has been canceled.
//   - evt - an *event.E that is being evaluated.
//   - hr - an *http.Request containing the information about the current
//     connection.
//   - origin - the address of the client.
//   - authedPubkey - the public key, if authed, of the client for this
//     connection.
//
// Return Values:
//
//   - accept - returns true if the event is accepted.
//   - notice - if it is not accepted,
//     a message in the form of `machine-readable-prefix: reason for
//     error/blocked/rate-limited/etc`
//   - afterSave - a closure to run after the event has been stored.
//
// Expected behaviour:
//
// This function checks whether the client has permission to store the event,
// and if they don't, returns false and some kind of error message. If they do,
// the event is forwarded to the database to be stored and indexed.
func (r *Relay) AcceptEvent(
	c context.T, evt *event.E, hr *http.Request,
	origin string, authedPubkey []byte,
) (accept bool, notice string, afterSave func()) {
	accept = true
	return
}

// AcceptFilter checks the provided filter against a set of accepted filters and
// determines if the filter should be accepted or not.
//
// Parameters:
//
//   - c - a context.T for signalling if the task has been canceled.
//   - hr - an *http.Request containing the information about the current
//     connection.
//   - f - a *filter.S that needs to be checked.
//   - authedPubkey - the public key, if authed, of the client for this
//     connection.
//
// Return Values:
//
//   - allowed - the filtered filter if it is accepted or nil otherwise
//   - ok - true if the filter is accepted, false otherwise
//   - modified - a boolean indicating whether the filter was modified
func (r *Relay) AcceptFilter(
	c context.T, hr *http.Request, f *filter.S,
	authedPubkey []byte,
) (allowed *filter.S, ok bool, modified bool) {
	allowed = f
	ok = true
	return
}

// AcceptReq checks an event and determines whether the event should be
// accepted and if the client has the authority to submit it.
//
// Parameters:
//
//   - c - a context.T for signalling if the task has been canceled.
//   - hr - an *http.Request containing the information about the current
//     connection.
//   - id - the ID of the request.
//   - ff - a set of filters that have been applied to the event, represented
//     as a filters.T object.
//   - authedPubkey - the public key, if authed, of the client for this
//     connection.
//
// Return Values:
//
//   - allowed - a filters.T object representing the accepted filters.
//   - ok - true if the event is accepted, false otherwise.
//   - modified - a boolean indicating whether the filters were modified.
func (r *Relay) AcceptReq(
	c context.T, hr *http.Request, id []byte,
	ff *filters.T, authedPubkey []byte,
) (allowed *filters.T, ok bool, modified bool) {
	allowed = ff
	ok = true
	return
}
