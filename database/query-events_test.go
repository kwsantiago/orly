package database

import (
	"bufio"
	"bytes"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/event"
	"orly.dev/event/examples"
	"orly.dev/filter"
	"orly.dev/kind"
	"orly.dev/kinds"
	"orly.dev/tag"
	"orly.dev/tags"
	"orly.dev/timestamp"
	"os"
	"testing"
)

func TestQueryEvents(t *testing.T) {
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

	// Test QueryEvents with an ID filter
	testEvent := events[3] // Using the same event as in other tests
	
	evs, err := db.QueryEvents(
		ctx, &filter.F{
			Ids: tag.New(testEvent.Id),
		},
	)
	if err != nil {
		t.Fatalf("Failed to query events by ID: %v", err)
	}
	
	// Verify we got exactly one event
	if len(evs) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(evs))
	}
	
	// Verify it's the correct event
	if !bytes.Equal(evs[0].Id, testEvent.Id) {
		t.Fatalf("Event ID doesn't match. Got %x, expected %x", evs[0].Id, testEvent.Id)
	}
	
	// Test querying by kind
	testKind := kind.New(1) // Kind 1 is typically text notes
	kindFilter := kinds.New(testKind)
	
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Kinds: kindFilter,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query events by kind: %v", err)
	}
	
	// Verify we got results
	if len(evs) == 0 {
		t.Fatal("Expected events with kind 1, but got none")
	}
	
	// Verify all events have the correct kind
	for i, ev := range evs {
		if ev.Kind.K != testKind.K {
			t.Fatalf("Event %d has incorrect kind. Got %d, expected %d", i, ev.Kind.K, testKind.K)
		}
	}
	
	// Test querying by author
	authorFilter := tag.New(events[1].Pubkey)
	
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Authors: authorFilter,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query events by author: %v", err)
	}
	
	// Verify we got results
	if len(evs) == 0 {
		t.Fatal("Expected events from author, but got none")
	}
	
	// Verify all events have the correct author
	for i, ev := range evs {
		if !bytes.Equal(ev.Pubkey, events[1].Pubkey) {
			t.Fatalf("Event %d has incorrect author. Got %x, expected %x", 
				i, ev.Pubkey, events[1].Pubkey)
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
	
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Since: sinceTime,
			Until: untilTime,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query events by time range: %v", err)
	}
	
	// Verify we got results
	if len(evs) == 0 {
		t.Fatal("Expected events in time range, but got none")
	}
	
	// Verify all events are within the time range
	for i, ev := range evs {
		if ev.CreatedAt.V < sinceTime.V || ev.CreatedAt.V > untilTime.V {
			t.Fatalf("Event %d is outside the time range. Got %d, expected between %d and %d", 
				i, ev.CreatedAt.V, sinceTime.V, untilTime.V)
		}
	}
	
	// Find an event with tags to use for testing
	var testTagEvent *event.E
	for _, ev := range events {
		if ev.Tags != nil && ev.Tags.Len() > 0 {
			// Find a tag with at least 2 elements and first element of length 1
			for _, tag := range ev.Tags.ToSliceOfTags() {
				if tag.Len() >= 2 && len(tag.B(0)) == 1 {
					testTagEvent = ev
					break
				}
			}
			if testTagEvent != nil {
				break
			}
		}
	}
	
	if testTagEvent != nil {
		// Get the first tag with at least 2 elements and first element of length 1
		var testTag *tag.T
		for _, tag := range testTagEvent.Tags.ToSliceOfTags() {
			if tag.Len() >= 2 && len(tag.B(0)) == 1 {
				testTag = tag
				break
			}
		}
		
		// Create a tags filter with the test tag
		tagsFilter := tags.New(testTag)
		
		evs, err = db.QueryEvents(
			ctx, &filter.F{
				Tags: tagsFilter,
			},
		)
		if err != nil {
			t.Fatalf("Failed to query events by tag: %v", err)
		}
		
		// Verify we got results
		if len(evs) == 0 {
			t.Fatal("Expected events with tag, but got none")
		}
		
		// Verify all events have the tag
		for i, ev := range evs {
			var hasTag bool
			for _, tag := range ev.Tags.ToSliceOfTags() {
				if tag.Len() >= 2 && len(tag.B(0)) == 1 {
					if bytes.Equal(tag.B(0), testTag.B(0)) && 
					   bytes.Equal(tag.B(1), testTag.B(1)) {
						hasTag = true
						break
					}
				}
			}
			if !hasTag {
				t.Fatalf("Event %d does not have the expected tag", i)
			}
		}
	}
}