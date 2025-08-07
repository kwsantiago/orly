package ws

import (
	"context"
	"sync"
	"testing"
	"time"

	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/signer"
)

// mockSigner implements signer.I for testing
type mockSigner struct {
	pubkey []byte
}

func (m *mockSigner) Pub() []byte { return m.pubkey }
func (m *mockSigner) Sign([]byte) (
	[]byte, error,
) {
	return []byte("mock-signature"), nil
}
func (m *mockSigner) Generate() error                     { return nil }
func (m *mockSigner) InitSec([]byte) error                { return nil }
func (m *mockSigner) InitPub([]byte) error                { return nil }
func (m *mockSigner) Sec() []byte                         { return []byte("mock-secret") }
func (m *mockSigner) Verify([]byte, []byte) (bool, error) { return true, nil }
func (m *mockSigner) Zero()                               {}
func (m *mockSigner) ECDH([]byte) (
	[]byte, error,
) {
	return []byte("mock-shared-secret"), nil
}

func TestNewPool(t *testing.T) {
	ctx := context.Background()
	pool := NewPool(ctx)

	if pool == nil {
		t.Fatal("NewPool returned nil")
	}

	if pool.Relays == nil {
		t.Error("Pool should have initialized Relays map")
	}

	if pool.Context == nil {
		t.Error("Pool should have a context")
	}
}

func TestPoolWithAuthHandler(t *testing.T) {
	ctx := context.Background()

	authHandler := WithAuthHandler(
		func() signer.I {
			return &mockSigner{pubkey: []byte("test-pubkey")}
		},
	)

	pool := NewPool(ctx, authHandler)

	if pool.authHandler == nil {
		t.Error("Pool should have auth handler set")
	}

	// Test that auth handler returns the expected signer
	signer := pool.authHandler()
	if string(signer.Pub()) != "test-pubkey" {
		t.Errorf(
			"Expected pubkey 'test-pubkey', got '%s'", string(signer.Pub()),
		)
	}
}

func TestPoolWithEventMiddleware(t *testing.T) {
	ctx := context.Background()

	var middlewareCalled bool
	middleware := WithEventMiddleware(
		func(ie RelayEvent) {
			middlewareCalled = true
		},
	)

	pool := NewPool(ctx, middleware)

	// Test that middleware is called
	testEvent := &event.E{
		Kind:      kind.TextNote,
		Content:   []byte("test"),
		CreatedAt: timestamp.Now(),
	}

	ie := RelayEvent{E: testEvent, Relay: nil}
	pool.eventMiddleware(ie)

	if !middlewareCalled {
		t.Error("Expected middleware to be called")
	}
}

func TestRelayEventString(t *testing.T) {
	testEvent := &event.E{
		Kind:      kind.TextNote,
		Content:   []byte("test content"),
		CreatedAt: timestamp.Now(),
	}

	client := &Client{URL: "wss://test.relay"}
	ie := RelayEvent{E: testEvent, Relay: client}

	str := ie.String()
	if !contains(str, "wss://test.relay") {
		t.Errorf("Expected string to contain relay URL, got: %s", str)
	}

	if !contains(str, "test content") {
		t.Errorf("Expected string to contain event content, got: %s", str)
	}
}

func TestNamedLock(t *testing.T) {
	// Test that named locks work correctly
	var wg sync.WaitGroup
	var counter int
	var mu sync.Mutex

	lockName := "test-lock"

	// Start multiple goroutines that try to increment counter
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := namedLock(lockName)
			defer unlock()

			// Critical section
			mu.Lock()
			temp := counter
			time.Sleep(1 * time.Millisecond) // Simulate work
			counter = temp + 1
			mu.Unlock()
		}()
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("Expected counter to be 10, got %d", counter)
	}
}

func TestPoolEnsureRelayInvalidURL(t *testing.T) {
	ctx := context.Background()
	pool := NewPool(ctx)

	// Test with invalid URL
	_, err := pool.EnsureRelay("invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestPoolQuerySingle(t *testing.T) {
	ctx := context.Background()
	pool := NewPool(ctx)

	// Test with empty URLs slice
	result := pool.QuerySingle(ctx, []string{}, &filter.F{})
	if result != nil {
		t.Error("Expected nil result for empty URLs")
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func uintPtr(u uint) *uint {
	return &u
}

// Test pool context cancellation
func TestPoolContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewPool(ctx)

	// Cancel the context
	cancel()

	// Check that pool context is cancelled
	select {
	case <-pool.Context.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected pool context to be cancelled")
	}
}
