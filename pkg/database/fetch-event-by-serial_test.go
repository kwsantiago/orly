package database

import (
	"bufio"
	"bytes"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/event/examples"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"os"
	"testing"
)

func TestFetchEventBySerial(t *testing.T) {
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

		eventCount++
	}

	// Check for scanner errors
	if err = scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	t.Logf("Successfully saved %d events to the database", eventCount)

	// Instead of trying to find a valid serial directly, let's use QueryForIds
	// which is known to work from the other tests
	testEvent := events[3] // Using the same event as in other tests

	// Use QueryForIds to get the IdPkTs for this event
	var sers types.Uint40s
	sers, err = db.QueryForSerials(
		ctx, &filter.F{
			Ids: tag.New(testEvent.Id),
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for Ids: %v", err)
	}

	// Verify we got exactly one result
	if len(sers) != 1 {
		t.Fatalf("Expected 1 IdPkTs, got %d", len(sers))
	}

	// Fetch the event by serial
	fetchedEvent, err := db.FetchEventBySerial(sers[0])
	if err != nil {
		t.Fatalf("Failed to fetch event by serial: %v", err)
	}

	// Verify the fetched event is not nil
	if fetchedEvent == nil {
		t.Fatal("Expected fetched event to be non-nil, but got nil")
	}

	// Verify the fetched event has the same ID as the original event
	if !bytes.Equal(fetchedEvent.Id, testEvent.Id) {
		t.Fatalf(
			"Fetched event ID doesn't match original event ID. Got %x, expected %x",
			fetchedEvent.Id, testEvent.Id,
		)
	}

	// Verify other event properties match
	if fetchedEvent.Kind.K != testEvent.Kind.K {
		t.Fatalf(
			"Fetched event kind doesn't match. Got %d, expected %d",
			fetchedEvent.Kind.K, testEvent.Kind.K,
		)
	}

	if !bytes.Equal(fetchedEvent.Pubkey, testEvent.Pubkey) {
		t.Fatalf(
			"Fetched event pubkey doesn't match. Got %x, expected %x",
			fetchedEvent.Pubkey, testEvent.Pubkey,
		)
	}

	if fetchedEvent.CreatedAt.V != testEvent.CreatedAt.V {
		t.Fatalf(
			"Fetched event created_at doesn't match. Got %d, expected %d",
			fetchedEvent.CreatedAt.V, testEvent.CreatedAt.V,
		)
	}

	// Test with a non-existent serial
	nonExistentSerial := new(types.Uint40)
	err = nonExistentSerial.Set(uint64(0xFFFFFFFFFF)) // Max value
	if err != nil {
		t.Fatalf("Failed to create non-existent serial: %v", err)
	}

	// This should return an error since the serial doesn't exist
	fetchedEvent, err = db.FetchEventBySerial(nonExistentSerial)
	if err == nil {
		t.Fatal("Expected error for non-existent serial, but got nil")
	}

	// The fetched event should be nil
	if fetchedEvent != nil {
		t.Fatalf(
			"Expected nil event for non-existent serial, but got: %v",
			fetchedEvent,
		)
	}
}
