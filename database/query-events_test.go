package database

import (
	"bufio"
	"bytes"
	"fmt"
	"orly.dev/encoders/event"
	"orly.dev/encoders/event/examples"
	"orly.dev/encoders/filter"
	"orly.dev/encoders/hex"
	"orly.dev/encoders/kind"
	"orly.dev/encoders/kinds"
	"orly.dev/encoders/tag"
	"orly.dev/encoders/tags"
	"orly.dev/encoders/timestamp"
	"orly.dev/utils/chk"
	"orly.dev/utils/context"
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
		t.Fatalf(
			"Event ID doesn't match. Got %x, expected %x", evs[0].Id,
			testEvent.Id,
		)
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
			t.Fatalf(
				"Event %d has incorrect kind. Got %d, expected %d", i,
				ev.Kind.K, testKind.K,
			)
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
			t.Fatalf(
				"Event %d has incorrect author. Got %x, expected %x",
				i, ev.Pubkey, events[1].Pubkey,
			)
		}
	}

	// Test querying for replaced events by ID
	// Create a replaceable event
	replaceableEvent := event.New()
	replaceableEvent.Kind = kind.ProfileMetadata // Kind 0 is replaceable
	replaceableEvent.Pubkey = events[0].Pubkey   // Use the same pubkey as an existing event
	replaceableEvent.CreatedAt = new(timestamp.T)
	replaceableEvent.CreatedAt.V = timestamp.Now().V - 7200 // 2 hours ago
	replaceableEvent.Content = []byte("Original profile")
	replaceableEvent.Tags = tags.New()

	// Save the replaceable event
	if _, _, err = db.SaveEvent(ctx, replaceableEvent); err != nil {
		t.Fatalf("Failed to save replaceable event: %v", err)
	}

	// Create a newer version of the replaceable event
	newerEvent := event.New()
	newerEvent.Kind = kind.ProfileMetadata      // Same kind
	newerEvent.Pubkey = replaceableEvent.Pubkey // Same pubkey
	newerEvent.CreatedAt = new(timestamp.T)
	newerEvent.CreatedAt.V = timestamp.Now().V - 3600 // 1 hour ago (newer than the original)
	newerEvent.Content = []byte("Updated profile")
	newerEvent.Tags = tags.New()

	// Save the newer event
	if _, _, err = db.SaveEvent(ctx, newerEvent); err != nil {
		t.Fatalf("Failed to save newer event: %v", err)
	}

	// Query for the original event by ID
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Ids: tag.New(replaceableEvent.Id),
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for replaced event by ID: %v", err)
	}

	// Verify we got exactly one event
	if len(evs) != 1 {
		t.Fatalf(
			"Expected 1 event when querying for replaced event by ID, got %d",
			len(evs),
		)
	}

	// Verify it's the original event
	if !bytes.Equal(evs[0].Id, replaceableEvent.Id) {
		t.Fatalf(
			"Event ID doesn't match when querying for replaced event. Got %x, expected %x",
			evs[0].Id, replaceableEvent.Id,
		)
	}

	// Query for all events of this kind and pubkey
	kindFilter = kinds.New(kind.ProfileMetadata)
	authorFilter = tag.New(replaceableEvent.Pubkey)

	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Kinds:   kindFilter,
			Authors: authorFilter,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for replaceable events: %v", err)
	}

	// Verify we got only one event (the latest one)
	if len(evs) != 1 {
		t.Fatalf(
			"Expected 1 event when querying for replaceable events, got %d",
			len(evs),
		)
	}

	// Verify it's the newer event
	if !bytes.Equal(evs[0].Id, newerEvent.Id) {
		t.Fatalf(
			"Event ID doesn't match when querying for replaceable events. Got %x, expected %x",
			evs[0].Id, newerEvent.Id,
		)
	}

	// Test deletion events
	// Create a deletion event that references the replaceable event
	deletionEvent := event.New()
	deletionEvent.Kind = kind.Deletion             // Kind 5 is deletion
	deletionEvent.Pubkey = replaceableEvent.Pubkey // Same pubkey as the event being deleted
	deletionEvent.CreatedAt = new(timestamp.T)
	deletionEvent.CreatedAt.V = timestamp.Now().V // Current time
	deletionEvent.Content = []byte("Deleting the replaceable event")
	deletionEvent.Tags = tags.New()

	// Add an e-tag referencing the replaceable event
	deletionEvent.Tags = deletionEvent.Tags.AppendTags(
		tag.New([]byte{'e'}, []byte(hex.Enc(replaceableEvent.Id))),
	)

	// Save the deletion event
	if _, _, err = db.SaveEvent(ctx, deletionEvent); err != nil {
		t.Fatalf("Failed to save deletion event: %v", err)
	}

	// Query for all events of this kind and pubkey again
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Kinds:   kindFilter,
			Authors: authorFilter,
		},
	)
	if err != nil {
		t.Fatalf(
			"Failed to query for replaceable events after deletion: %v", err,
		)
	}

	// Verify we still get the newer event (deletion should only affect the original event)
	if len(evs) != 1 {
		t.Fatalf(
			"Expected 1 event when querying for replaceable events after deletion, got %d",
			len(evs),
		)
	}

	// Verify it's still the newer event
	if !bytes.Equal(evs[0].Id, newerEvent.Id) {
		t.Fatalf(
			"Event ID doesn't match after deletion. Got %x, expected %x",
			evs[0].Id, newerEvent.Id,
		)
	}

	// Query for the original event by ID
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Ids: tag.New(replaceableEvent.Id),
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for deleted event by ID: %v", err)
	}

	// Verify we still get the original event when querying by ID
	if len(evs) != 1 {
		t.Fatalf(
			"Expected 1 event when querying for deleted event by ID, got %d",
			len(evs),
		)
	}

	// Verify it's the original event
	if !bytes.Equal(evs[0].Id, replaceableEvent.Id) {
		t.Fatalf(
			"Event ID doesn't match when querying for deleted event by ID. Got %x, expected %x",
			evs[0].Id, replaceableEvent.Id,
		)
	}

	// Create a parameterized replaceable event
	paramEvent := event.New()
	paramEvent.Kind = kind.New(30000)    // Kind 30000+ is parameterized replaceable
	paramEvent.Pubkey = events[0].Pubkey // Use the same pubkey as an existing event
	paramEvent.CreatedAt = new(timestamp.T)
	paramEvent.CreatedAt.V = timestamp.Now().V - 7200 // 2 hours ago
	paramEvent.Content = []byte("Original parameterized event")
	paramEvent.Tags = tags.New()

	// Add a d-tag
	paramEvent.Tags = paramEvent.Tags.AppendTags(
		tag.New([]byte{'d'}, []byte("test-d-tag")),
	)

	// Save the parameterized replaceable event
	if _, _, err = db.SaveEvent(ctx, paramEvent); err != nil {
		t.Fatalf("Failed to save parameterized replaceable event: %v", err)
	}

	// Create a deletion event that references the parameterized replaceable event using an a-tag
	paramDeletionEvent := event.New()
	paramDeletionEvent.Kind = kind.Deletion       // Kind 5 is deletion
	paramDeletionEvent.Pubkey = paramEvent.Pubkey // Same pubkey as the event being deleted
	paramDeletionEvent.CreatedAt = new(timestamp.T)
	paramDeletionEvent.CreatedAt.V = timestamp.Now().V // Current time
	paramDeletionEvent.Content = []byte("Deleting the parameterized replaceable event")
	paramDeletionEvent.Tags = tags.New()

	// Add an a-tag referencing the parameterized replaceable event
	// Format: kind:pubkey:d-tag
	aTagValue := fmt.Sprintf(
		"%d:%s:%s",
		paramEvent.Kind.K,
		hex.Enc(paramEvent.Pubkey),
		"test-d-tag",
	)

	paramDeletionEvent.Tags = paramDeletionEvent.Tags.AppendTags(
		tag.New([]byte{'a'}, []byte(aTagValue)),
	)

	// Save the parameterized deletion event
	if _, _, err = db.SaveEvent(ctx, paramDeletionEvent); err != nil {
		t.Fatalf("Failed to save parameterized deletion event: %v", err)
	}

	// Query for all events of this kind and pubkey
	paramKindFilter := kinds.New(paramEvent.Kind)
	paramAuthorFilter := tag.New(paramEvent.Pubkey)

	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Kinds:   paramKindFilter,
			Authors: paramAuthorFilter,
		},
	)
	if err != nil {
		t.Fatalf(
			"Failed to query for parameterized replaceable events after deletion: %v",
			err,
		)
	}

	// Verify we get no events (since the only one was deleted)
	if len(evs) != 0 {
		t.Fatalf(
			"Expected 0 events when querying for deleted parameterized replaceable events, got %d",
			len(evs),
		)
	}

	// Query for the parameterized event by ID
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Ids: tag.New(paramEvent.Id),
		},
	)
	if err != nil {
		t.Fatalf(
			"Failed to query for deleted parameterized event by ID: %v", err,
		)
	}

	// Verify we still get the event when querying by ID
	if len(evs) != 1 {
		t.Fatalf(
			"Expected 1 event when querying for deleted parameterized event by ID, got %d",
			len(evs),
		)
	}

	// Verify it's the correct event
	if !bytes.Equal(evs[0].Id, paramEvent.Id) {
		t.Fatalf(
			"Event ID doesn't match when querying for deleted parameterized event by ID. Got %x, expected %x",
			evs[0].Id, paramEvent.Id,
		)
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
			t.Fatalf(
				"Event %d is outside the time range. Got %d, expected between %d and %d",
				i, ev.CreatedAt.V, sinceTime.V, untilTime.V,
			)
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
