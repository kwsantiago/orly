package database

import (
	"bufio"
	"bytes"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/event"
	"orly.dev/event/examples"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
	"orly.dev/log"
	"orly.dev/tag"
	"os"
	"testing"
)

func TestQueryForIds(t *testing.T) {
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
		if _, _, err = db.SaveEvent(ctx, ev); err != nil {
			t.Fatalf("Failed to save event #%d: %v", eventCount+1, err)
		}

		eventCount++
	}

	// Check for scanner errors
	if err = scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	t.Logf("Successfully saved %d events to the database", eventCount)

	var idTsPk []store.IdTsPk
	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Ids: tag.New(events[3].Id),
		},
	)
	log.I.S(idTsPk)
	// idTsPk, err = db.QueryForIds(
	// 	ctx, &filter.F{
	// 		Authors: tag.New(events[0].Pubkey),
	// 	},
	// )
	// log.I.S(idTsPk)
}
