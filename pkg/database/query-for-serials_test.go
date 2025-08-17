package database

import (
	"bufio"
	"bytes"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/event/examples"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"os"
	"testing"
)

func TestQueryForSerials(t *testing.T) {
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

	var events []*event.E
	var eventSerials = make(map[string]*types.Uint40) // Map event ID (hex) to serial

	// Process each event
	for scanner.Scan() {
		chk.E(scanner.Err())
		b := scanner.Bytes()
		ev := event.New()

		// Unmarshal the event
		if _, err = ev.Unmarshal(b); chk.E(err) {
			t.Fatal(err)
		}

		events = append(events, ev)

		// Save the event to the database
		if _, _, err = db.SaveEvent(ctx, ev, false, nil); err != nil {
			t.Fatalf("Failed to save event #%d: %v", eventCount+1, err)
		}

		// Get the serial for this event
		serial, err := db.GetSerialById(ev.ID)
		if err != nil {
			t.Fatalf(
				"Failed to get serial for event #%d: %v", eventCount+1, err,
			)
		}

		if serial != nil {
			eventSerials[string(ev.ID)] = serial
		}

		eventCount++
	}

	// Check for scanner errors
	if err = scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	t.Logf("Successfully saved %d events to the database", eventCount)

	// Test QueryForSerials with an ID filter
	testEvent := events[3] // Using the same event as in other tests

	serials, err := db.QueryForSerials(
		ctx, &filter.F{
			Ids: tag.New(testEvent.ID),
		},
	)
	if err != nil {
		t.Fatalf("Failed to query serials by ID: %v", err)
	}

	// Verify we got exactly one serial
	if len(serials) != 1 {
		t.Fatalf("Expected 1 serial, got %d", len(serials))
	}

	// Verify the serial corresponds to the correct event
	// Fetch the event using the serial
	ev, err := db.FetchEventBySerial(serials[0])
	if err != nil {
		t.Fatalf("Failed to fetch event for serial: %v", err)
	}

	if !utils.FastEqual(ev.ID, testEvent.ID) {
		t.Fatalf(
			"Event ID doesn't match. Got %x, expected %x",
			ev.ID, testEvent.ID,
		)
	}

	// Test querying by kind
	testKind := kind.New(1) // Kind 1 is typically text notes
	kindFilter := kinds.New(testKind)

	serials, err = db.QueryForSerials(
		ctx, &filter.F{
			Kinds: kindFilter,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query serials by kind: %v", err)
	}

	// Verify we got results
	if len(serials) == 0 {
		t.Fatal("Expected serials for events with kind 1, but got none")
	}

	// Verify the serials correspond to events with the correct kind
	for i, serial := range serials {
		// Fetch the event using the serial
		ev, err := db.FetchEventBySerial(serial)
		if err != nil {
			t.Fatalf("Failed to fetch event for serial %d: %v", i, err)
		}

		if ev.Kind.K != testKind.K {
			t.Fatalf(
				"Event %d has incorrect kind. Got %d, expected %d",
				i, ev.Kind.K, testKind.K,
			)
		}
	}

	// Test querying by author
	authorFilter := tag.New(events[1].Pubkey)

	serials, err = db.QueryForSerials(
		ctx, &filter.F{
			Authors: authorFilter,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query serials by author: %v", err)
	}

	// Verify we got results
	if len(serials) == 0 {
		t.Fatal("Expected serials for events from author, but got none")
	}

	// Verify the serials correspond to events with the correct author
	for i, serial := range serials {
		// Fetch the event using the serial
		ev, err := db.FetchEventBySerial(serial)
		if err != nil {
			t.Fatalf("Failed to fetch event for serial %d: %v", i, err)
		}

		if !utils.FastEqual(ev.Pubkey, events[1].Pubkey) {
			t.Fatalf(
				"Event %d has incorrect author. Got %x, expected %x",
				i, ev.Pubkey, events[1].Pubkey,
			)
		}
	}

	// Test querying by time range
	// Use the timestamp from the middle event as a reference
	middleIndex := len(events) / 2
	middleEvent := events[middleIndex]

	// Create a timestamp range that includes events before and after the middle event
	sinceTime := new(timestamp.T)
	sinceTime.V = middleEvent.CreatedAt.V - 3600 // 1 hour before middle event

	untilTime := new(timestamp.T)
	untilTime.V = middleEvent.CreatedAt.V + 3600 // 1 hour after middle event

	serials, err = db.QueryForSerials(
		ctx, &filter.F{
			Since: sinceTime,
			Until: untilTime,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query serials by time range: %v", err)
	}

	// Verify we got results
	if len(serials) == 0 {
		t.Fatal("Expected serials for events in time range, but got none")
	}

	// Verify the serials correspond to events within the time range
	for i, serial := range serials {
		// Fetch the event using the serial
		ev, err := db.FetchEventBySerial(serial)
		if err != nil {
			t.Fatalf("Failed to fetch event for serial %d: %v", i, err)
		}

		if ev.CreatedAt.V < sinceTime.V || ev.CreatedAt.V > untilTime.V {
			t.Fatalf(
				"Event %d is outside the time range. Got %d, expected between %d and %d",
				i, ev.CreatedAt.V, sinceTime.V, untilTime.V,
			)
		}
	}
}
