package openapi

import (
	"net/http"
	"testing"
	"time"

	"orly.dev/pkg/app/config"

	"orly.dev/pkg/app/relay/publish"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/interfaces/store"
	ctx "orly.dev/pkg/utils/context"
)

// mockServer implements the server.I interface for testing
type mockServer struct {
	authRequired bool
	context      ctx.T
}

// Implement the methods needed for our tests
func (m *mockServer) AuthRequired() bool {
	return m.authRequired
}

func (m *mockServer) Context() ctx.T {
	return m.context
}

func (m *mockServer) Publisher() *publish.S {
	return nil // Not used in our tests
}

// Stub implementations for the rest of the server.I interface
func (m *mockServer) AcceptEvent(
	c ctx.T, ev *event.E, hr *http.Request, authedPubkey []byte,
	remote string,
) (accept bool, notice string, afterSave func()) {
	return true, "", nil
}

func (m *mockServer) AcceptReq(
	c ctx.T, hr *http.Request, f *filters.T,
	authedPubkey []byte, remote string,
) (allowed *filters.T, accept bool, modified bool) {
	return f, true, false
}

func (m *mockServer) AddEvent(
	c ctx.T, rl relay.I, ev *event.E, hr *http.Request, origin string,
	pubkeys [][]byte,
) (accepted bool, message []byte) {
	return true, nil
}

func (m *mockServer) AdminAuth(
	r *http.Request, remote string, tolerance ...time.Duration,
) (authed bool, pubkey []byte) {
	return false, nil
}

func (m *mockServer) UserAuth(
	r *http.Request, remote string, tolerance ...time.Duration,
) (authed bool, pubkey []byte, super bool) {
	return false, nil, false
}

func (m *mockServer) Publish(c ctx.T, evt *event.E) (err error) {
	return nil
}

func (m *mockServer) Relay() relay.I {
	return nil
}

func (m *mockServer) Shutdown() {}

func (m *mockServer) Storage() store.I {
	return nil
}

func (m *mockServer) PublicReadable() bool {
	return true
}

func (m *mockServer) ServiceURL(req *http.Request) (s string) {
	return ""
}

func (m *mockServer) OwnersPubkeys() (pks [][]byte) {
	return nil
}

func (m *mockServer) Config() (c *config.C) {
	return
}

// TestPublisherFunctionality tests the listen/subscribe/unsubscribe and publisher functionality
func TestPublisherFunctionality(t *testing.T) {
	// Create a context with cancel function
	testCtx, cancel := ctx.Cancel(ctx.Bg())
	defer cancel()

	// Create a mock server
	mockServer := &mockServer{
		authRequired: false,
		context:      testCtx,
	}

	// Create a publisher
	publisher := NewPublisher(mockServer)

	// Test 1: Register a listener
	t.Run(
		"RegisterListener", func(t *testing.T) {
			// Create a receiver channel
			receiver := make(DeliverChan, 32)

			// Create a listener
			listener := &H{
				Id:        "test-listener",
				Receiver:  receiver,
				FilterMap: make(map[string]*filter.F),
				New:       true,
			}

			// Register the listener
			publisher.Receive(listener)

			// Verify the listener was registered
			if _, ok := publisher.ListenMap["test-listener"]; !ok {
				t.Errorf("Listener was not registered")
			}
		},
	)

	// Test 2: Add a subscription
	t.Run(
		"AddSubscription", func(t *testing.T) {
			// Create a filter
			f := &filter.F{}

			// Create a subscription
			subscription := &H{
				Id: "test-listener",
				FilterMap: map[string]*filter.F{
					"test-subscription": f,
				},
			}

			// Add the subscription
			publisher.Receive(subscription)

			// Verify the subscription was added
			listener, ok := publisher.ListenMap["test-listener"]
			if !ok {
				t.Errorf("Listener not found")
				return
			}

			if _, ok := listener.FilterMap["test-subscription"]; !ok {
				t.Errorf("Subscription was not added")
			}
		},
	)

	// Test 3: Deliver an event
	t.Run(
		"DeliverEvent", func(t *testing.T) {
			// Create an event that matches the filter
			ev := &event.E{
				Kind:      kind.TextNote,
				CreatedAt: timestamp.Now(),
			}

			// Deliver the event
			publisher.Deliver(ev)

			// Get the listener
			listener, ok := publisher.ListenMap["test-listener"]
			if !ok {
				t.Errorf("Listener not found")
				return
			}

			// Verify the event was received
			select {
			case receivedEv := <-listener.Receiver:
				if receivedEv.Event != ev {
					t.Errorf("Received event does not match delivered event")
				}
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Event was not received within timeout")
			}
		},
	)

	// Test 4: Unsubscribe
	t.Run(
		"Unsubscribe", func(t *testing.T) {
			// Create a new listener first since the previous one was removed
			receiver := make(DeliverChan, 32)
			listener := &H{
				Id:        "test-listener",
				Receiver:  receiver,
				FilterMap: make(map[string]*filter.F),
				New:       true,
			}
			publisher.Receive(listener)

			// Add a subscription
			subscription := &H{
				Id: "test-listener",
				FilterMap: map[string]*filter.F{
					"test-subscription": &filter.F{},
				},
			}
			publisher.Receive(subscription)

			// Create an unsubscribe message
			unsubscribe := &H{
				Id: "test-listener",
				FilterMap: map[string]*filter.F{
					"test-subscription": nil,
				},
				Cancel: true,
			}

			// Unsubscribe
			publisher.Receive(unsubscribe)

			// Verify the subscription was removed
			listener, ok := publisher.ListenMap["test-listener"]
			if !ok {
				t.Errorf("Listener was removed, but should still exist")
				return
			}
			if _, ok := listener.FilterMap["test-subscription"]; ok {
				t.Errorf("Subscription was not removed")
			}
		},
	)

	// Test 5: Remove listener
	t.Run(
		"RemoveListener", func(t *testing.T) {
			// Create a remove listener message
			removeListener := &H{
				Id:     "test-listener",
				Cancel: true,
			}

			// Remove the listener
			publisher.Receive(removeListener)

			// Verify the listener was removed
			if _, ok := publisher.ListenMap["test-listener"]; ok {
				t.Errorf("Listener was not removed")
			}
		},
	)

	// Test 6: Edge case - Unsubscribe non-existent subscription
	t.Run(
		"UnsubscribeNonExistentSubscription", func(t *testing.T) {
			// Create a new listener first
			receiver := make(DeliverChan, 32)
			listener := &H{
				Id:        "test-listener-2",
				Receiver:  receiver,
				FilterMap: make(map[string]*filter.F),
				New:       true,
			}
			publisher.Receive(listener)

			// Add a subscription to ensure the listener has at least one subscription
			subscription := &H{
				Id: "test-listener-2",
				FilterMap: map[string]*filter.F{
					"existing-subscription": &filter.F{},
				},
			}
			publisher.Receive(subscription)

			// Create an unsubscribe message for a non-existent subscription
			unsubscribe := &H{
				Id: "test-listener-2",
				FilterMap: map[string]*filter.F{
					"non-existent-subscription": nil,
				},
				Cancel: true,
			}

			// Unsubscribe
			publisher.Receive(unsubscribe)

			// Verify the listener still exists (since it still has one subscription)
			if _, ok := publisher.ListenMap["test-listener-2"]; !ok {
				t.Errorf("Listener was removed, but should still exist since it has other subscriptions")
			}

			// Verify the existing subscription is still there
			listener, ok := publisher.ListenMap["test-listener-2"]
			if !ok {
				t.Errorf("Listener not found")
				return
			}
			if _, ok := listener.FilterMap["existing-subscription"]; !ok {
				t.Errorf("Existing subscription was removed")
			}
		},
	)

	// Test 7: Edge case - Deliver event with authentication required
	t.Run(
		"DeliverEventWithAuthRequired", func(t *testing.T) {
			// Set auth required to true
			mockServer.authRequired = true

			// Create a new listener with pubkey
			receiver := make(DeliverChan, 32)
			listener := &H{
				Id:        "test-listener-3",
				Receiver:  receiver,
				FilterMap: make(map[string]*filter.F),
				Pubkey:    []byte("test-pubkey"),
				New:       true,
			}
			publisher.Receive(listener)

			// Add a subscription
			subscription := &H{
				Id: "test-listener-3",
				FilterMap: map[string]*filter.F{
					"test-subscription-3": &filter.F{},
				},
			}
			publisher.Receive(subscription)

			// Create an event with a different pubkey and a privileged kind
			ev := &event.E{
				Kind:      kind.EncryptedDirectMessage,
				Pubkey:    []byte("different-pubkey"),
				Tags:      tags.New(), // Initialize empty tags
				CreatedAt: timestamp.Now(),
			}

			// Deliver the event
			publisher.Deliver(ev)

			// Verify the event was not received (due to auth check)
			select {
			case <-listener.Receiver:
				t.Errorf("Event was received, but should have been blocked by auth check")
			case <-time.After(100 * time.Millisecond):
				// This is expected - no event should be received
			}

			// Reset auth required
			mockServer.authRequired = false
		},
	)

	// Test 8: Filter matching - Events are only delivered to listeners with matching filters
	t.Run(
		"FilterMatching", func(t *testing.T) {
			// Create two listeners with different filters
			receiver1 := make(DeliverChan, 32)
			listener1 := &H{
				Id:        "test-listener-filter-1",
				Receiver:  receiver1,
				FilterMap: make(map[string]*filter.F),
				New:       true,
			}
			publisher.Receive(listener1)

			receiver2 := make(DeliverChan, 32)
			listener2 := &H{
				Id:        "test-listener-filter-2",
				Receiver:  receiver2,
				FilterMap: make(map[string]*filter.F),
				New:       true,
			}
			publisher.Receive(listener2)

			// Add different filters to each listener
			// First filter matches events with kind.TextNote
			filter1 := &filter.F{
				Kinds: kinds.New(kind.TextNote),
			}
			subscription1 := &H{
				Id: "test-listener-filter-1",
				FilterMap: map[string]*filter.F{
					"filter-subscription-1": filter1,
				},
			}
			publisher.Receive(subscription1)

			// Second filter matches events with kind.EncryptedDirectMessage
			filter2 := &filter.F{
				Kinds: kinds.New(kind.EncryptedDirectMessage),
			}
			subscription2 := &H{
				Id: "test-listener-filter-2",
				FilterMap: map[string]*filter.F{
					"filter-subscription-2": filter2,
				},
			}
			publisher.Receive(subscription2)

			// Create an event that matches only the first filter
			ev := &event.E{
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				CreatedAt: timestamp.Now(),
			}

			// Deliver the event
			publisher.Deliver(ev)

			// Verify the event was received by the first listener
			select {
			case receivedEv := <-receiver1:
				if receivedEv.Event != ev {
					t.Errorf("Received event does not match delivered event")
				}
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Event was not received by first listener within timeout")
			}

			// Verify the event was NOT received by the second listener
			select {
			case <-receiver2:
				t.Errorf("Event was received by second listener, but should not have matched its filter")
			case <-time.After(100 * time.Millisecond):
				// This is expected - no event should be received
			}
		},
	)
}
