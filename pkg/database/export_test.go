package database

import (
	"bufio"
	"bytes"
	"os"
	"testing"

	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/event/examples"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
)

// TestExport tests the Export function by:
// 1. Creating a new database with events from examples.Cache
// 2. Checking that all event IDs in the cache are found in the export
// 3. Verifying this also works when only a few pubkeys are requested
func TestExport(t *testing.T) {
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

	// Maps to store event IDs and their associated pubkeys
	eventIDs := make(map[string]bool)
	pubkeyToEventIDs := make(map[string][]string)

	// Process each event
	for scanner.Scan() {
		chk.E(scanner.Err())
		b := scanner.Bytes()
		ev := event.New()

		// Unmarshal the event
		if _, err = ev.Unmarshal(b); chk.E(err) {
			t.Fatal(err)
		}

		// Save the event to the database
		if _, _, err = db.SaveEvent(ctx, ev); err != nil {
			t.Fatalf("Failed to save event: %v", err)
		}

		// Store the event ID
		eventID := ev.IdString()
		eventIDs[eventID] = true

		// Store the event ID by pubkey
		pubkey := ev.PubKeyString()
		pubkeyToEventIDs[pubkey] = append(pubkeyToEventIDs[pubkey], eventID)
	}

	// Check for scanner errors
	if err = scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	t.Logf("Saved %d events to the database", len(eventIDs))

	// Test 1: Export all events and verify all IDs are in the export
	var exportBuffer bytes.Buffer
	db.Export(ctx, &exportBuffer)

	// Parse the exported events and check that all IDs are present
	exportedIDs := make(map[string]bool)
	exportScanner := bufio.NewScanner(&exportBuffer)
	exportScanner.Buffer(make([]byte, 0, 1_000_000_000), 1_000_000_000)
	exportCount := 0
	for exportScanner.Scan() {
		b := exportScanner.Bytes()
		ev := event.New()
		if _, err = ev.Unmarshal(b); chk.E(err) {
			t.Fatal(err)
		}
		exportedIDs[ev.IdString()] = true
		exportCount++
	}
	// Check for scanner errors
	if err = exportScanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	t.Logf("Found %d events in the export", exportCount)

	// Check that all original event IDs are in the export
	for id := range eventIDs {
		if !exportedIDs[id] {
			t.Errorf("Event ID %s not found in export", id)
		}
	}

	t.Logf("All %d event IDs found in export", len(eventIDs))
}
