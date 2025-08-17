package database

import (
	"bufio"
	"bytes"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/event/examples"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"os"
	"testing"
)

func TestQueryForKindsTags(t *testing.T) {
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

	// Find an event with tags to use for testing
	var testEvent *event.E
	for _, ev := range events {
		if ev.Tags != nil && ev.Tags.Len() > 0 {
			// Find a tag with at least 2 elements and first element of length 1
			for _, tag := range ev.Tags.ToSliceOfTags() {
				if tag.Len() >= 2 && len(tag.B(0)) == 1 {
					testEvent = ev
					break
				}
			}
			if testEvent != nil {
				break
			}
		}
	}

	if testEvent == nil {
		t.Skip("No suitable event with tags found for testing")
	}

	// Get the first tag with at least 2 elements and first element of length 1
	var testTag *tag.T
	for _, tag := range testEvent.Tags.ToSliceOfTags() {
		if tag.Len() >= 2 && len(tag.B(0)) == 1 {
			testTag = tag
			break
		}
	}

	// Test querying by kind and tag
	var idTsPk []store.IdPkTs

	// Use the kind from the test event
	testKind := testEvent.Kind
	kindFilter := kinds.New(testKind)

	// Create a tags filter with the test tag
	tagsFilter := tags.New(testTag)

	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Kinds: kindFilter,
			Tags:  tagsFilter,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for kinds and tags: %v", err)
	}

	// Verify we got results
	if len(idTsPk) == 0 {
		t.Fatal("did not find any events with the specified kind and tag")
	}

	// Verify the results have the correct kind and tag
	for i, result := range idTsPk {
		// Find the event with this ID
		var found bool
		for _, ev := range events {
			if utils.FastEqual(result.Id, ev.ID) {
				found = true
				if ev.Kind.K != testKind.K {
					t.Fatalf(
						"result %d has incorrect kind, got %d, expected %d",
						i, ev.Kind.K, testKind.K,
					)
				}

				// Check if the event has the tag we're looking for
				var hasTag bool
				for _, tag := range ev.Tags.ToSliceOfTags() {
					if tag.Len() >= 2 && len(tag.B(0)) == 1 {
						if utils.FastEqual(
							tag.B(0), testTag.B(0),
						) && utils.FastEqual(tag.B(1), testTag.B(1)) {
							hasTag = true
							break
						}
					}
				}

				if !hasTag {
					t.Fatalf(
						"result %d does not have the expected tag",
						i,
					)
				}

				break
			}
		}
		if !found {
			t.Fatalf("result %d with ID %x not found in events", i, result.Id)
		}
	}
}
