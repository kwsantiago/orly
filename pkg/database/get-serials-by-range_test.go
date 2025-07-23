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

func TestGetSerialsByRange(t *testing.T) {
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
		if _, _, err = db.SaveEvent(ctx, ev, false); err != nil {
			t.Fatalf("Failed to save event #%d: %v", eventCount+1, err)
		}

		// Get the serial for this event
		serial, err := db.GetSerialById(ev.Id)
		if err != nil {
			t.Fatalf(
				"Failed to get serial for event #%d: %v", eventCount+1, err,
			)
		}

		if serial != nil {
			eventSerials[string(ev.Id)] = serial
		}

		eventCount++
	}

	// Check for scanner errors
	if err = scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	t.Logf("Successfully saved %d events to the database", eventCount)

	// Test GetSerialsByRange with a time range filter
	// Use the timestamp from the middle event as a reference
	middleIndex := len(events) / 2
	middleEvent := events[middleIndex]

	// Create a timestamp range that includes events before and after the middle event
	sinceTime := new(timestamp.T)
	sinceTime.V = middleEvent.CreatedAt.V - 3600 // 1 hour before middle event

	untilTime := new(timestamp.T)
	untilTime.V = middleEvent.CreatedAt.V + 3600 // 1 hour after middle event

	// Create a filter with the time range
	timeFilter := &filter.F{
		Since: sinceTime,
		Until: untilTime,
	}

	// Get the indexes from the filter
	ranges, err := GetIndexesFromFilter(timeFilter)
	if err != nil {
		t.Fatalf("Failed to get indexes from filter: %v", err)
	}

	// Verify we got at least one range
	if len(ranges) == 0 {
		t.Fatal("Expected at least one range from filter, but got none")
	}

	// Test GetSerialsByRange with the first range
	serials, err := db.GetSerialsByRange(ranges[0])
	if err != nil {
		t.Fatalf("Failed to get serials by range: %v", err)
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

	// Test GetSerialsByRange with a kind filter
	testKind := kind.New(1) // Kind 1 is typically text notes
	kindFilter := &filter.F{
		Kinds: kinds.New(testKind),
	}

	// Get the indexes from the filter
	ranges, err = GetIndexesFromFilter(kindFilter)
	if err != nil {
		t.Fatalf("Failed to get indexes from filter: %v", err)
	}

	// Verify we got at least one range
	if len(ranges) == 0 {
		t.Fatal("Expected at least one range from filter, but got none")
	}

	// Test GetSerialsByRange with the first range
	serials, err = db.GetSerialsByRange(ranges[0])
	if err != nil {
		t.Fatalf("Failed to get serials by range: %v", err)
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

	// Test GetSerialsByRange with an author filter
	authorFilter := &filter.F{
		Authors: tag.New(events[1].Pubkey),
	}

	// Get the indexes from the filter
	ranges, err = GetIndexesFromFilter(authorFilter)
	if err != nil {
		t.Fatalf("Failed to get indexes from filter: %v", err)
	}

	// Verify we got at least one range
	if len(ranges) == 0 {
		t.Fatal("Expected at least one range from filter, but got none")
	}

	// Test GetSerialsByRange with the first range
	serials, err = db.GetSerialsByRange(ranges[0])
	if err != nil {
		t.Fatalf("Failed to get serials by range: %v", err)
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

		if !bytes.Equal(ev.Pubkey, events[1].Pubkey) {
			t.Fatalf(
				"Event %d has incorrect author. Got %x, expected %x",
				i, ev.Pubkey, events[1].Pubkey,
			)
		}
	}
}
