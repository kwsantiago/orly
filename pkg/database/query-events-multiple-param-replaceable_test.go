package database

import (
	"bytes"
	"fmt"
	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/utils/chk"
	"os"
	"testing"
)

// TestMultipleParameterizedReplaceableEvents tests that when multiple parameterized
// replaceable events with the same pubkey, kind, and d-tag exist, only the newest one
// is returned in query results.
func TestMultipleParameterizedReplaceableEvents(t *testing.T) {
	db, _, ctx, cancel, tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir) // Clean up after the test
	defer cancel()
	defer db.Close()

	sign := new(p256k.Signer)
	if err := sign.Generate(); chk.E(err) {
		t.Fatal(err)
	}

	// Create a base parameterized replaceable event
	baseEvent := event.New()
	baseEvent.Kind = kind.New(30000) // Kind 30000+ is parameterized replaceable
	baseEvent.CreatedAt = new(timestamp.T)
	baseEvent.CreatedAt.V = timestamp.Now().V - 7200 // 2 hours ago
	baseEvent.Content = []byte("Original parameterized event")
	baseEvent.Tags = tags.New()
	// Add a d-tag
	baseEvent.Tags = baseEvent.Tags.AppendTags(
		tag.New([]byte{'d'}, []byte("test-d-tag")),
	)
	baseEvent.Sign(sign)

	// Save the base parameterized replaceable event
	if _, _, err := db.SaveEvent(ctx, baseEvent, false); err != nil {
		t.Fatalf("Failed to save base parameterized replaceable event: %v", err)
	}

	// Create a newer parameterized replaceable event with the same pubkey, kind, and d-tag
	newerEvent := event.New()
	newerEvent.Kind = kind.New(30000) // Same kind
	newerEvent.CreatedAt = new(timestamp.T)
	newerEvent.CreatedAt.V = timestamp.Now().V - 3600 // 1 hour ago (newer than base event)
	newerEvent.Content = []byte("Newer parameterized event")
	newerEvent.Tags = tags.New()
	// Add the same d-tag
	newerEvent.Tags = newerEvent.Tags.AppendTags(
		tag.New([]byte{'d'}, []byte("test-d-tag")),
	)
	newerEvent.Sign(sign)

	// Save the newer parameterized replaceable event
	if _, _, err := db.SaveEvent(ctx, newerEvent, false); err != nil {
		t.Fatalf(
			"Failed to save newer parameterized replaceable event: %v", err,
		)
	}

	// Create an even newer parameterized replaceable event with the same pubkey, kind, and d-tag
	newestEvent := event.New()
	newestEvent.Kind = kind.New(30000) // Same kind
	newestEvent.CreatedAt = new(timestamp.T)
	newestEvent.CreatedAt.V = timestamp.Now().V // Current time (newest)
	newestEvent.Content = []byte("Newest parameterized event")
	newestEvent.Tags = tags.New()
	// Add the same d-tag
	newestEvent.Tags = newestEvent.Tags.AppendTags(
		tag.New([]byte{'d'}, []byte("test-d-tag")),
	)
	newestEvent.Sign(sign)

	// Save the newest parameterized replaceable event
	if _, _, err := db.SaveEvent(ctx, newestEvent, false); err != nil {
		t.Fatalf(
			"Failed to save newest parameterized replaceable event: %v", err,
		)
	}

	// Query for all events of this kind and pubkey
	paramKindFilter := kinds.New(baseEvent.Kind)
	paramAuthorFilter := tag.New(baseEvent.Pubkey)

	evs, err := db.QueryEvents(
		ctx, &filter.F{
			Kinds:   paramKindFilter,
			Authors: paramAuthorFilter,
		},
	)
	if err != nil {
		t.Fatalf(
			"Failed to query for parameterized replaceable events: %v", err,
		)
	}

	// Print debug info about the returned events
	fmt.Printf("Debug: Got %d events\n", len(evs))
	for i, ev := range evs {
		fmt.Printf(
			"Debug: Event %d: kind=%d, pubkey=%s, created_at=%d, content=%s\n",
			i, ev.Kind.K, hex.Enc(ev.Pubkey), ev.CreatedAt.V, ev.Content,
		)
		dTag := ev.Tags.GetFirst(tag.New([]byte{'d'}))
		if dTag != nil && dTag.Len() > 1 {
			fmt.Printf("Debug: Event %d: d-tag=%s\n", i, dTag.Value())
		}
	}

	// Verify we get exactly one event (the newest one)
	if len(evs) != 1 {
		t.Fatalf(
			"Expected 1 event when querying for parameterized replaceable events, got %d",
			len(evs),
		)
	}

	// Verify it's the newest event
	if !bytes.Equal(evs[0].ID, newestEvent.ID) {
		t.Fatalf(
			"Event ID doesn't match the newest event. Got %x, expected %x",
			evs[0].ID, newestEvent.ID,
		)
	}

	// Verify the content is from the newest event
	if string(evs[0].Content) != string(newestEvent.Content) {
		t.Fatalf(
			"Event content doesn't match the newest event. Got %s, expected %s",
			evs[0].Content, newestEvent.Content,
		)
	}

	// Query for the base event by ID
	evs, err = db.QueryEvents(
		ctx, &filter.F{
			Ids: tag.New(baseEvent.ID),
		},
	)
	if err != nil {
		t.Fatalf("Failed to query for base event by ID: %v", err)
	}

	// Verify we can still get the base event when querying by ID
	if len(evs) != 1 {
		t.Fatalf(
			"Expected 1 event when querying for base event by ID, got %d",
			len(evs),
		)
	}

	// Verify it's the base event
	if !bytes.Equal(evs[0].ID, baseEvent.ID) {
		t.Fatalf(
			"Event ID doesn't match when querying for base event by ID. Got %x, expected %x",
			evs[0].ID, baseEvent.ID,
		)
	}
}
