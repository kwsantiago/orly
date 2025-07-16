package socketapi

import (
	"errors"
	"orly.dev/encoders/envelopes/closedenvelope"
	"orly.dev/interfaces/server"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
	"orly.dev/utils/normalize"
	"orly.dev/utils/pointers"

	"github.com/dgraph-io/badger/v4"

	"orly.dev/encoders/envelopes/eoseenvelope"
	"orly.dev/encoders/envelopes/eventenvelope"
	"orly.dev/encoders/envelopes/reqenvelope"
	"orly.dev/encoders/event"
	"orly.dev/utils/context"
)

// HandleReq processes a raw request, parses its envelope, validates filters,
// and interacts with the server storage and subscription mechanisms to query
// events or manage subscriptions.
//
// Parameters:
//
//   - c: A context object used for managing deadlines, cancellation signals,
//     and other request-scoped values.
//
//   - req: A byte slice representing the raw request data to be processed.
//
//   - srv: An interface representing the server, providing access to storage
//     and subscription management.
//
// Return values:
//
//   - r: A byte slice containing the response or error message generated
//     during processing.
//
// Expected behavior:
//
// The method parses and validates the incoming request envelope, querying
// events from the server storage based on filters provided. It sends results
// through the associated subscription or writes error messages to the listener.
// If the subscription should be canceled due to completed query results, it
// generates and sends a closure envelope.
func (a *A) HandleReq(
	c context.T, req []byte, srv server.S,
) (r []byte) {
	log.I.F("REQ:\n%s", req)
	sto := srv.Storage()
	var err error
	var rem []byte
	env := reqenvelope.New()
	if rem, err = env.Unmarshal(req); chk.E(err) {
		return normalize.Error.F(err.Error())
	}
	if len(rem) > 0 {
		log.I.F("extra '%s'", rem)
	}
	allowed := env.Filters
	var events event.S
	for _, f := range allowed.F {
		// var i uint
		if pointers.Present(f.Limit) {
			if *f.Limit == 0 {
				continue
			}
		}
		if events, err = sto.QueryEvents(c, f); chk.E(err) {
			log.E.F("eventstore: %v", err)
			if errors.Is(err, badger.ErrDBClosed) {
				return
			}
			continue
		}
		// write out the events to the socket
		for _, ev := range events {
			var res *eventenvelope.Result
			if res, err = eventenvelope.NewResultWith(
				env.Subscription.T,
				ev,
			); chk.E(err) {
				return
			}
			if err = res.Write(a.Listener); chk.E(err) {
				return
			}
		}
	}
	if err = eoseenvelope.NewFrom(env.Subscription).Write(a.Listener); chk.E(err) {
		return
	}
	receiver := make(event.C, 32)
	cancel := true
	// if the query was for just Ids, we know there cannot be any more results,
	// so cancel the subscription.
	for _, f := range allowed.F {
		if f.Ids.Len() < 1 {
			cancel = false
			break
		}
		// also, if we received the limit number of events, subscription ded
		if pointers.Present(f.Limit) {
			if len(events) < int(*f.Limit) {
				cancel = false
			}
		}
		if !cancel {
			break
		}
	}
	if !cancel {
		srv.Publisher().Receive(
			&W{
				Listener: a.Listener,
				Id:       env.Subscription.String(),
				Receiver: receiver,
				Filters:  env.Filters,
			},
		)
	} else {
		if err = closedenvelope.NewFrom(
			env.Subscription, nil,
		).Write(a.Listener); chk.E(err) {
			return
		}
	}
	return
}
