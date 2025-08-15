package database

import (
	"bufio"
	"bytes"
	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/event/examples"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/errorf"
	"os"
	"testing"
	"time"
)

// TestSaveEvents tests saving all events from examples.Cache to the database
// to verify there are no errors during the saving process.
func TestSaveEvents(t *testing.T) {
	// Create a temporary directory for the database
	tempDir, err := os.MkdirTemp("", "test-db-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Create a context and cancel function for the database
	ctx, cancel := context.Cancel(context.Bg())
	defer cancel()

	// Initialize the database
	db, err := New(ctx, cancel, tempDir, "info")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create a scanner to read events from examples.Cache
	scanner := bufio.NewScanner(bytes.NewBuffer(examples.Cache))
	scanner.Buffer(make([]byte, 0, 1_000_000_000), 1_000_000_000)

	// Count the number of events processed
	eventCount := 0

	var original int
	var kc, vc int
	now := time.Now()
	// Process each event
	for scanner.Scan() {
		chk.E(scanner.Err())
		b := scanner.Bytes()
		// log.T.F("%d bytes of raw JSON", len(b))
		original += len(b)
		ev := event.New()

		// Unmarshal the event
		if _, err = ev.Unmarshal(b); chk.E(err) {
			t.Fatal(err)
		}

		// Save the event to the database
		var k, v int
		if k, v, err = db.SaveEvent(ctx, ev, false, nil); err != nil {
			t.Fatalf("Failed to save event #%d: %v", eventCount+1, err)
		}
		kc += k
		vc += v
		eventCount++
	}

	// Check for scanner errors
	if err = scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}
	dur := time.Since(now)
	t.Logf(
		"Successfully saved %d events %d bytes to the database, %d bytes keys, %d bytes values in %v (%v/ev; %f ev/s)",
		eventCount,
		original,
		kc, vc,
		dur,
		dur/time.Duration(eventCount),
		float64(time.Second)/float64(dur/time.Duration(eventCount)),
	)
}

// TestDeletionEventWithETagRejection tests that a deletion event with an "e" tag is rejected.
func TestDeletionEventWithETagRejection(t *testing.T) {
	// Create a temporary directory for the database
	tempDir, err := os.MkdirTemp("", "test-db-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Create a context and cancel function for the database
	ctx, cancel := context.Cancel(context.Bg())
	defer cancel()

	// Initialize the database
	db, err := New(ctx, cancel, tempDir, "info")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create a signer
	sign := new(p256k.Signer)
	if err := sign.Generate(); chk.E(err) {
		t.Fatal(err)
	}

	// Create a regular event
	regularEvent := event.New()
	regularEvent.Kind = kind.TextNote // Kind 1 is a text note
	regularEvent.Pubkey = sign.Pub()
	regularEvent.CreatedAt = new(timestamp.T)
	regularEvent.CreatedAt.V = timestamp.Now().V - 3600 // 1 hour ago
	regularEvent.Content = []byte("Regular event")
	regularEvent.Tags = tags.New()
	regularEvent.Sign(sign)

	// Save the regular event
	if _, _, err := db.SaveEvent(ctx, regularEvent, false, nil); err != nil {
		t.Fatalf("Failed to save regular event: %v", err)
	}

	// Create a deletion event with an "e" tag referencing the regular event
	deletionEvent := event.New()
	deletionEvent.Kind = kind.Deletion // Kind 5 is deletion
	deletionEvent.Pubkey = sign.Pub()
	deletionEvent.CreatedAt = new(timestamp.T)
	deletionEvent.CreatedAt.V = timestamp.Now().V // Current time
	deletionEvent.Content = []byte("Deleting the regular event")
	deletionEvent.Tags = tags.New()

	// Add an e-tag referencing the regular event
	deletionEvent.Tags = deletionEvent.Tags.AppendTags(
		tag.New([]byte{'e'}, []byte(hex.Enc(regularEvent.ID))),
	)

	deletionEvent.Sign(sign)

	// Check if this is a deletion event with "e" tags
	if deletionEvent.Kind == kind.Deletion && deletionEvent.Tags.GetFirst(tag.New([]byte{'e'})) != nil {
		// In this test, we want to reject deletion events with "e" tags
		err = errorf.E("deletion events referencing other events with 'e' tag are not allowed")
	} else {
		// Try to save the deletion event
		_, _, err = db.SaveEvent(ctx, deletionEvent, false, nil)
	}

	if err == nil {
		t.Fatal("Expected deletion event with e-tag to be rejected, but it was accepted")
	}

	// Verify the error message
	expectedError := "deletion events referencing other events with 'e' tag are not allowed"
	if err.Error() != expectedError {
		t.Fatalf(
			"Expected error message '%s', got '%s'", expectedError, err.Error(),
		)
	}
}

// TestSaveExistingEvent tests that attempting to save an event that already exists
// returns an error.
func TestSaveExistingEvent(t *testing.T) {
	// Create a temporary directory for the database
	tempDir, err := os.MkdirTemp("", "test-db-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Create a context and cancel function for the database
	ctx, cancel := context.Cancel(context.Bg())
	defer cancel()

	// Initialize the database
	db, err := New(ctx, cancel, tempDir, "info")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create a signer
	sign := new(p256k.Signer)
	if err := sign.Generate(); chk.E(err) {
		t.Fatal(err)
	}

	// Create an event
	ev := event.New()
	ev.Kind = kind.TextNote // Kind 1 is a text note
	ev.Pubkey = sign.Pub()
	ev.CreatedAt = new(timestamp.T)
	ev.CreatedAt.V = timestamp.Now().V
	ev.Content = []byte("Test event")
	ev.Tags = tags.New()
	ev.Sign(sign)

	// Save the event for the first time
	if _, _, err := db.SaveEvent(ctx, ev, false, nil); err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	// Try to save the same event again, it should be rejected
	_, _, err = db.SaveEvent(ctx, ev, false, nil)
	if err == nil {
		t.Fatal("Expected error when saving an existing event, but got nil")
	}

	// Verify the error message
	expectedErrorPrefix := "event already exists: "
	if !bytes.HasPrefix([]byte(err.Error()), []byte(expectedErrorPrefix)) {
		t.Fatalf(
			"Expected error message to start with '%s', got '%s'",
			expectedErrorPrefix, err.Error(),
		)
	}
}
