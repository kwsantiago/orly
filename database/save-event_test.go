package database

import (
	"bufio"
	"bytes"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/event"
	"orly.dev/event/examples"
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
		if k, v, err = db.SaveEvent(ctx, ev); err != nil {
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
