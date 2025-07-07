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
	"orly.dev/kind"
	"orly.dev/kinds"
	"orly.dev/tag"
	"orly.dev/tags"
	"orly.dev/timestamp"
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

	var idTsPk []store.IdPkTs
	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Ids: tag.New(events[3].Id),
		},
	)
	if len(idTsPk) != 1 {
		t.Fatal("did not find expected event")
	}
	if !bytes.Equal(idTsPk[0].Id, events[3].Id) {
		t.Fatalf(
			"did not find expected event, got %0x, expected %0x",
			idTsPk[0].Id, events[3].Id,
		)
	}
	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Authors: tag.New(events[1].Pubkey),
		},
	)
	if len(idTsPk) != 5 {
		t.Fatalf(
			"got unexpected number of results, expect 5, got %d",
			len(idTsPk),
		)
	}
	if !bytes.Equal(idTsPk[0].Id, events[5474].Id) {
		t.Fatalf(
			"failed to get expected event, got %0x, expected %0x", idTsPk[0].Id,
			events[5474].Id,
		)
	}
	if !bytes.Equal(idTsPk[1].Id, events[272].Id) {
		t.Fatalf(
			"failed to get expected event, got %0x, expected %0x", idTsPk[1].Id,
			events[272].Id,
		)
	}
	if !bytes.Equal(idTsPk[2].Id, events[1].Id) {
		t.Fatalf(
			"failed to get expected event, got %0x, expected %0x", idTsPk[2].Id,
			events[1].Id,
		)
	}
	if !bytes.Equal(idTsPk[3].Id, events[80].Id) {
		t.Fatalf(
			"failed to get expected event, got %0x, expected %0x", idTsPk[3].Id,
			events[80].Id,
		)
	}
	if !bytes.Equal(idTsPk[4].Id, events[123].Id) {
		t.Fatalf(
			"failed to get expected event, got %0x, expected %0x", idTsPk[4].Id,
			events[123].Id,
		)
	}

	// Test querying by kind
	// Find an event with a specific kind
	testKind := kind.New(1) // Kind 1 is typically text notes
	kindFilter := kinds.New(testKind)

	idTsPk, err = db.QueryForIds(
		ctx, &filter.F{
			Kinds: kindFilter,
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for kinds: %v", err)
	}

	// Verify we got results
	if len(idTsPk) == 0 {
		t.Fatal("did not find any events with the specified kind")
	}

	// Verify the results have the correct kind
	for i, result := range idTsPk {
		// Find the event with this ID
		var found bool
		for _, ev := range events {
			if bytes.Equal(result.Id, ev.Id) {
				found = true
				if ev.Kind.K != testKind.K {
					t.Fatalf(
						"result %d has incorrect kind, got %d, expected %d",
						i, ev.Kind.K, testKind.K,
					)
				}
				break
			}
		}
		if !found {
			t.Fatalf("result %d with ID %x not found in events", i, result.Id)
		}
	}

	// Test querying by tag
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

	if testEvent != nil {
		// Get the first tag with at least 2 elements and first element of length 1
		var testTag *tag.T
		for _, tag := range testEvent.Tags.ToSliceOfTags() {
			if tag.Len() >= 2 && len(tag.B(0)) == 1 {
				testTag = tag
				break
			}
		}

		// Create a tags filter with the test tag
		tagsFilter := tags.New(testTag)

		idTsPk, err = db.QueryForIds(
			ctx, &filter.F{
				Tags: tagsFilter,
			},
		)
		if err != nil {
			t.Fatalf("Failed to query for tags: %v", err)
		}

		// Verify we got results
		if len(idTsPk) == 0 {
			t.Fatal("did not find any events with the specified tag")
		}

		// Verify the results have the correct tag
		for i, result := range idTsPk {
			// Find the event with this ID
			var found bool
			for _, ev := range events {
				if bytes.Equal(result.Id, ev.Id) {
					found = true

					// Check if the event has the tag we're looking for
					var hasTag bool
					for _, tag := range ev.Tags.ToSliceOfTags() {
						if tag.Len() >= 2 && len(tag.B(0)) == 1 {
							if bytes.Equal(
								tag.B(0), testTag.B(0),
							) && bytes.Equal(tag.B(1), testTag.B(1)) {
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
				t.Fatalf(
					"result %d with ID %x not found in events", i, result.Id,
				)
			}
		}

		// Test querying by kind and author
		idTsPk, err = db.QueryForIds(
			ctx, &filter.F{
				Kinds:   kindFilter,
				Authors: tag.New(events[1].Pubkey),
			},
		)
		if err != nil {
			t.Fatalf("Failed to query for kinds and authors: %v", err)
		}

		// Verify we got results
		if len(idTsPk) > 0 {
			// Verify the results have the correct kind and author
			for i, result := range idTsPk {
				// Find the event with this ID
				var found bool
				for _, ev := range events {
					if bytes.Equal(result.Id, ev.Id) {
						found = true
						if ev.Kind.K != testKind.K {
							t.Fatalf(
								"result %d has incorrect kind, got %d, expected %d",
								i, ev.Kind.K, testKind.K,
							)
						}
						if !bytes.Equal(ev.Pubkey, events[1].Pubkey) {
							t.Fatalf(
								"result %d has incorrect author, got %x, expected %x",
								i, ev.Pubkey, events[1].Pubkey,
							)
						}
						break
					}
				}
				if !found {
					t.Fatalf(
						"result %d with ID %x not found in events", i,
						result.Id,
					)
				}
			}
		}

		// Test querying by kind and tag
		idTsPk, err = db.QueryForIds(
			ctx, &filter.F{
				Kinds: kinds.New(testEvent.Kind),
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
				if bytes.Equal(result.Id, ev.Id) {
					found = true
					if ev.Kind.K != testEvent.Kind.K {
						t.Fatalf(
							"result %d has incorrect kind, got %d, expected %d",
							i, ev.Kind.K, testEvent.Kind.K,
						)
					}

					// Check if the event has the tag we're looking for
					var hasTag bool
					for _, tag := range ev.Tags.ToSliceOfTags() {
						if tag.Len() >= 2 && len(tag.B(0)) == 1 {
							if bytes.Equal(
								tag.B(0), testTag.B(0),
							) && bytes.Equal(tag.B(1), testTag.B(1)) {
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
				t.Fatalf(
					"result %d with ID %x not found in events", i, result.Id,
				)
			}
		}

		// Test querying by kind, author, and tag
		idTsPk, err = db.QueryForIds(
			ctx, &filter.F{
				Kinds:   kinds.New(testEvent.Kind),
				Authors: tag.New(testEvent.Pubkey),
				Tags:    tagsFilter,
			},
		)
		if err != nil {
			t.Fatalf("Failed to query for kinds, authors, and tags: %v", err)
		}

		// Verify we got results
		if len(idTsPk) == 0 {
			t.Fatal("did not find any events with the specified kind, author, and tag")
		}

		// Verify the results have the correct kind, author, and tag
		for i, result := range idTsPk {
			// Find the event with this ID
			var found bool
			for _, ev := range events {
				if bytes.Equal(result.Id, ev.Id) {
					found = true
					if ev.Kind.K != testEvent.Kind.K {
						t.Fatalf(
							"result %d has incorrect kind, got %d, expected %d",
							i, ev.Kind.K, testEvent.Kind.K,
						)
					}

					if !bytes.Equal(ev.Pubkey, testEvent.Pubkey) {
						t.Fatalf(
							"result %d has incorrect author, got %x, expected %x",
							i, ev.Pubkey, testEvent.Pubkey,
						)
					}

					// Check if the event has the tag we're looking for
					var hasTag bool
					for _, tag := range ev.Tags.ToSliceOfTags() {
						if tag.Len() >= 2 && len(tag.B(0)) == 1 {
							if bytes.Equal(
								tag.B(0), testTag.B(0),
							) && bytes.Equal(tag.B(1), testTag.B(1)) {
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
				t.Fatalf(
					"result %d with ID %x not found in events", i, result.Id,
				)
			}
		}

		// Test querying by author and tag
		idTsPk, err = db.QueryForIds(
			ctx, &filter.F{
				Authors: tag.New(testEvent.Pubkey),
				Tags:    tagsFilter,
			},
		)
		if err != nil {
			t.Fatalf("Failed to query for authors and tags: %v", err)
		}

		// Verify we got results
		if len(idTsPk) == 0 {
			t.Fatal("did not find any events with the specified author and tag")
		}

		// Verify the results have the correct author and tag
		for i, result := range idTsPk {
			// Find the event with this ID
			var found bool
			for _, ev := range events {
				if bytes.Equal(result.Id, ev.Id) {
					found = true

					if !bytes.Equal(ev.Pubkey, testEvent.Pubkey) {
						t.Fatalf(
							"result %d has incorrect author, got %x, expected %x",
							i, ev.Pubkey, testEvent.Pubkey,
						)
					}

					// Check if the event has the tag we're looking for
					var hasTag bool
					for _, tag := range ev.Tags.ToSliceOfTags() {
						if tag.Len() >= 2 && len(tag.B(0)) == 1 {
							if bytes.Equal(
								tag.B(0), testTag.B(0),
							) && bytes.Equal(tag.B(1), testTag.B(1)) {
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
				t.Fatalf(
					"result %d with ID %x not found in events", i, result.Id,
				)
			}
		}
	}

	// Test querying by created_at range
	// Use the timestamp from the middle event as a reference
	middleIndex := len(events) / 2
	middleEvent := events[middleIndex]

	// Create a timestamp range that includes events before and after the middle event
	sinceTime := new(timestamp.T)
	sinceTime.V = middleEvent.CreatedAt.V - 3600 // 1 hour before middle event

	untilTime := new(timestamp.T)
	untilTime.V = middleEvent.CreatedAt.V + 3600 // 1 hour after middle event

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

	// Verify the results exist in our events slice
	for i, result := range idTsPk {
		// Find the event with this ID
		var found bool
		for _, ev := range events {
			if bytes.Equal(result.Id, ev.Id) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("result %d with ID %x not found in events", i, result.Id)
		}
	}
}
