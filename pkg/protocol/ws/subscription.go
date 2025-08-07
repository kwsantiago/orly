package ws

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"orly.dev/pkg/encoders/envelopes/closeenvelope"
	"orly.dev/pkg/encoders/envelopes/reqenvelope"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/subscription"
	"orly.dev/pkg/utils/chk"
)

// Subscription represents a subscription to a relay.
type Subscription struct {
	counter int64
	id      string

	Relay   *Client
	Filters *filters.T

	// // for this to be treated as a COUNT and not a REQ this must be set
	// countResult chan CountEnvelope

	// the Events channel emits all EVENTs that come in a Subscription
	// will be closed when the subscription ends
	Events event.C
	mu     sync.Mutex

	// the EndOfStoredEvents channel gets closed when an EOSE comes for that subscription
	EndOfStoredEvents chan struct{}

	// the ClosedReason channel emits the reason when a CLOSED message is received
	ClosedReason chan string

	// Context will be .Done() when the subscription ends
	Context context.Context

	// // if it is not nil, checkDuplicate will be called for every event received
	// // if it returns true that event will not be processed further.
	// checkDuplicate func(id string, relay string) bool
	//
	// // if it is not nil, checkDuplicateReplaceable will be called for every event received
	// // if it returns true that event will not be processed further.
	// checkDuplicateReplaceable func(rk ReplaceableKey, ts Timestamp) bool

	match  func(*event.E) bool // this will be either Filters.Match or Filters.MatchIgnoringTimestampConstraints
	live   atomic.Bool
	eosed  atomic.Bool
	cancel context.CancelCauseFunc

	// this keeps track of the events we've received before the EOSE that we must dispatch before
	// closing the EndOfStoredEvents channel
	storedwg sync.WaitGroup
}

// SubscriptionOption is the type of the argument passed when instantiating relay connections.
// Some examples are WithLabel.
type SubscriptionOption interface {
	IsSubscriptionOption()
}

// WithLabel puts a label on the subscription (it is prepended to the automatic id) that is sent to relays.
type WithLabel string

func (_ WithLabel) IsSubscriptionOption() {}

// // WithCheckDuplicate sets checkDuplicate on the subscription
// type WithCheckDuplicate func(id, relay string) bool
//
// func (_ WithCheckDuplicate) IsSubscriptionOption() {}
//
// // WithCheckDuplicateReplaceable sets checkDuplicateReplaceable on the subscription
// type WithCheckDuplicateReplaceable func(rk ReplaceableKey, ts *timestamp.T) bool
//
// func (_ WithCheckDuplicateReplaceable) IsSubscriptionOption() {}

var (
	_ SubscriptionOption = (WithLabel)("")
	// _ SubscriptionOption = (WithCheckDuplicate)(nil)
	// _ SubscriptionOption = (WithCheckDuplicateReplaceable)(nil)
)

func (sub *Subscription) start() {
	<-sub.Context.Done()

	// the subscription ends once the context is canceled (if not already)
	sub.unsub(errors.New("context done on start()")) // this will set sub.live to false

	// do this so we don't have the possibility of closing the Events channel and then trying to send to it
	sub.mu.Lock()
	close(sub.Events)
	sub.mu.Unlock()
}

// GetID returns the subscription ID.
func (sub *Subscription) GetID() string { return sub.id }

func (sub *Subscription) dispatchEvent(evt *event.E) {
	added := false
	if !sub.eosed.Load() {
		sub.storedwg.Add(1)
		added = true
	}

	go func() {
		sub.mu.Lock()
		defer sub.mu.Unlock()

		if sub.live.Load() {
			select {
			case sub.Events <- evt:
			case <-sub.Context.Done():
			}
		}
		if added {
			sub.storedwg.Done()
		}
	}()
}

func (sub *Subscription) dispatchEose() {
	if sub.eosed.CompareAndSwap(false, true) {
		sub.match = sub.Filters.MatchIgnoringTimestampConstraints
		go func() {
			sub.storedwg.Wait()
			sub.EndOfStoredEvents <- struct{}{}
		}()
	}
}

// handleClosed handles the CLOSED message from a relay.
func (sub *Subscription) handleClosed(reason string) {
	go func() {
		sub.ClosedReason <- reason
		sub.live.Store(false) // set this so we don't send an unnecessary CLOSE to the relay
		sub.unsub(fmt.Errorf("CLOSED received: %s", reason))
	}()
}

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events and makes a new one.
func (sub *Subscription) Unsub() {
	sub.unsub(errors.New("Unsub() called"))
}

// unsub is the internal implementation of Unsub.
func (sub *Subscription) unsub(err error) {
	// cancel the context (if it's not canceled already)
	sub.cancel(err)

	// mark subscription as closed and send a CLOSE to the relay (naïve sync.Once implementation)
	if sub.live.CompareAndSwap(true, false) {
		sub.Close()
	}

	// remove subscription from our map
	sub.Relay.Subscriptions.Delete(sub.counter)
}

// Close just sends a CLOSE message. You probably want Unsub() instead.
func (sub *Subscription) Close() {
	if sub.Relay.IsConnected() {
		id, err := subscription.NewId(sub.id)
		if err != nil {
			return
		}
		closeMsg := closeenvelope.NewFrom(id)
		closeb := closeMsg.Marshal(nil)
		<-sub.Relay.Write(closeb)
	}
}

// Sub sets sub.Filters and then calls sub.Fire(ctx).
// The subscription will be closed if the context expires.
func (sub *Subscription) Sub(_ context.Context, ff *filters.T) {
	sub.Filters = ff
	sub.Fire()
}

// Fire sends the "REQ" command to the relay.
func (sub *Subscription) Fire() (err error) {
	// if sub.countResult == nil {
	req := reqenvelope.NewWithIdString(sub.id, sub.Filters)
	if req == nil {
		return fmt.Errorf("invalid ID or filters")
	}
	reqb := req.Marshal(nil)
	// } else
	// if len(sub.Filters) == 1 {
	// 	reqb, _ = CountEnvelope{sub.id, sub.Filters[0], nil, nil}.MarshalJSON()
	// } else {
	// 	return fmt.Errorf("unexpected sub configuration")
	sub.live.Store(true)
	if err = <-sub.Relay.Write(reqb); chk.E(err) {
		err = fmt.Errorf("failed to write: %w", err)
		sub.cancel(err)
	}
	return
}
