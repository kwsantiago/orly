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
// # Return Values
//
//   - string: the name of the application.
//
// # Expected behaviour
//
// This function simply returns the AppName field from the configuration.
func (r *Relay) Name() string { return r.C.AppName }

// Storage represents a persistence layer for Nostr events handled by a relay.
func (r *Relay) Storage() store.I { return r.Store }

// Init initializes and sets up the relay for Nostr events.
//
// #Return Values
//
//   - err: an error if any issues occurred during initialization.
//
// #Expected behaviour
//
// This function is responsible for setting up the relay, configuring it,
// and initializing the necessary components to handle Nostr events.
func (r *Relay) Init() (err error) {
	return nil
}

// AcceptEvent checks an event and determines whether the event should be
// accepted and if the client has the authority to submit it.
//
// # Parameters
//
//   - c - a context.T for signalling if the task has been canceled.
//
//   - evt - an *event.E that is being evaluated.
//
//   - hr - an *http.Request containing the information about the current
//     connection.
//
//   - origin - the address of the client.
//
//   - authedPubkey - the public key, if authed, of the client for this
//     connection.
//
// # Return Values
//
//   - accept - returns true if the event is accepted.
//
//   - notice - if it is not accepted, a message in the form of
//     `machine-readable-prefix: reason for error/blocked/rate-limited/etc`
//
//   - afterSave - a closure to run after the event has been stored.
//
// # Expected behaviour
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

// AcceptFilter checks if a filter is allowed based on authentication status and
// relay policies
//
// # Parameters
//
//   - c: Context for task cancellation.
//
//   - hr: HTTP request containing connection information.
//
//   - f: Filter to evaluate for acceptance.
//
//   - authedPubkey: Public key of authenticated client, if applicable.
//
// # Return values
//
//   - allowed: The filter if permitted; may be modified during processing.
//
//   - ok: Boolean indicating whether the filter is accepted.
//
//   - modified: Boolean indicating whether the filter was altered during
//     evaluation.
//
// # Expected behaviour
//
// The method evaluates whether the provided filter should be allowed based on
// authentication status and relay-specific rules. If permitted, returns the
// filter (possibly modified) and true for ok; otherwise returns nil or false
// for ok accordingly.
func (r *Relay) AcceptFilter(
	c context.T, hr *http.Request, f *filter.S,
	authedPubkey []byte,
) (allowed *filter.S, ok bool, modified bool) {
	allowed = f
	ok = true
	return
}

// AcceptReq evaluates whether the provided filters are allowed based on
// authentication status and relay policies for an incoming HTTP request.
//
// # Parameters
//
//   - c: Context for task cancellation.
//
//   - hr: HTTP request containing connection information.
//
//   - id: Identifier associated with the request.
//
//   - ff: Filters to evaluate for acceptance.
//
//   - authedPubkey: Public key of authenticated client, if applicable.
//
// # Return Values
//
//   - allowed: The filters if permitted; may be modified during processing.
//
//   - ok: Boolean indicating whether the filters are accepted.
//
//   - modified: Boolean indicating whether the filters were altered during
//     evaluation.
//
// # Expected Behaviour:
//
// The method evaluates whether the provided filters should be allowed based on
// authentication status and relay-specific rules. If permitted, returns the
// filters (possibly modified) and true for ok; otherwise returns nil or false
// for ok accordingly.
func (r *Relay) AcceptReq(
	c context.T, hr *http.Request, id []byte,
	ff *filters.T, authedPubkey []byte,
) (allowed *filters.T, ok bool, modified bool) {
	allowed = ff
	ok = true
	return
}
