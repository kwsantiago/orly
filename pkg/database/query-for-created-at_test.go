package database

import (
	"bufio"
	"bytes"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/event/examples"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"os"
	"testing"
)

func TestQueryForCreatedAt(t *testing.T) {
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
		if _, _, err = db.SaveEvent(ctx, ev, false, nil); err != nil {
			t.Fatalf("Failed to save event #%d: %v", eventCount+1, err)
		}

		eventCount++
	}

	// Check for scanner errors
	if err = scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	t.Logf("Successfully saved %d events to the database", eventCount)

	// Find a timestamp range that should include some events
	// Use the timestamp from the middle event as a reference
	middleIndex := len(events) / 2
	middleEvent := events[middleIndex]

	// Create a timestamp range that includes events before and after the middle event
	sinceTime := new(timestamp.T)
	sinceTime.V = middleEvent.CreatedAt.V - 3600 // 1 hour before middle event

	untilTime := new(timestamp.T)
	untilTime.V = middleEvent.CreatedAt.V + 3600 // 1 hour after middle event

	// Test querying by created_at range
	var idTsPk []store.IdPkTs

	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Since: sinceTime,
			Until: untilTime,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for created_at range: %v", err)
	}

	// Verify we got results
	if len(idTsPk) == 0 {
		t.Fatal("did not find any events in the specified time range")
	}

	// Verify the results exist in our events slice and are within the timestamp range
	for i, result := range idTsPk {
		// Find the event with this ID
		var found bool
		for _, ev := range events {
			if bytes.Equal(result.Id, ev.ID) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("result %d with ID %x not found in events", i, result.Id)
		}

		// Verify the timestamp is within the range
		if result.Ts < sinceTime.V || result.Ts > untilTime.V {
			t.Fatalf(
				"result %d with ID %x has timestamp %d outside the range [%d, %d]",
				i, result.Id, result.Ts, sinceTime.V, untilTime.V,
			)
		}
	}

	// Test with only Since
	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Since: sinceTime,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query with Since: %v", err)
	}

	// Verify we got results
	if len(idTsPk) == 0 {
		t.Fatal("did not find any events with Since filter")
	}

	// Verify the results exist in our events slice and are after the Since timestamp
	for i, result := range idTsPk {
		// Find the event with this ID
		var found bool
		for _, ev := range events {
			if bytes.Equal(result.Id, ev.ID) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("result %d with ID %x not found in events", i, result.Id)
		}

		// Verify the timestamp is after the Since timestamp
		if result.Ts < sinceTime.V {
			t.Fatalf(
				"result %d with ID %x has timestamp %d before the Since timestamp %d",
				i, result.Id, result.Ts, sinceTime.V,
			)
		}
	}

	// Test with only Until
	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Until: untilTime,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query with Until: %v", err)
	}

	// Verify we got results
	if len(idTsPk) == 0 {
		t.Fatal("did not find any events with Until filter")
	}

	// Verify the results exist in our events slice and are before the Until timestamp
	for i, result := range idTsPk {
		// Find the event with this ID
		var found bool
		for _, ev := range events {
			if bytes.Equal(result.Id, ev.ID) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("result %d with ID %x not found in events", i, result.Id)
		}

		// Verify the timestamp is before the Until timestamp
		if result.Ts > untilTime.V {
			t.Fatalf(
				"result %d with ID %x has timestamp %d after the Until timestamp %d",
				i, result.Id, result.Ts, untilTime.V,
			)
		}
	}
}
