package database

import (
	"testing"

	"not.realy.lol/event"
	"not.realy.lol/kind"
	"not.realy.lol/tag"
	"not.realy.lol/tags"
	"not.realy.lol/timestamp"
)

func TestGenerateIndexes(t *testing.T) {
	// Create a sample event with known fields
	ev := event.New()

	// Set event ID (32 bytes)
	ev.Id = make([]byte, 32)
	for i := 0; i < 32; i++ {
		ev.Id[i] = byte(i)
	}

	// Set pubkey (32 bytes)
	ev.Pubkey = make([]byte, 32)
	for i := 0; i < 32; i++ {
		ev.Pubkey[i] = byte(i + 32)
	}

	// Set created_at timestamp
	ev.CreatedAt = timestamp.New(1234567890)

	// Set kind
	ev.Kind = kind.New(1) // Set to kind 1 (short text note)

	// Set tags
	ev.Tags = tags.New()
	ev.Tags.AppendTags(tag.New("e", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"))
	ev.Tags.AppendTags(tag.New("p", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))

	// Set content
	ev.Content = []byte("This is a test event")

	// Set signature (not used in index generation, but included for completeness)
	ev.Sig = make([]byte, 64)

	// Generate indexes with a known serial number
	serial := uint64(42)
	indexes := GenerateIndexes(ev, serial)

	// Verify that the correct number of indexes were generated
	// The number depends on the implementation of GenerateIndexes
	// For the current implementation with 2 tags, we expect:
	// 1 Event index
	// 1 Id index
	// 1 IdPubkeyCreatedAt index
	// 1 CreatedAt index
	// 1 PubkeyCreatedAt index
	// 2 PubkeyTagCreatedAt indexes (one for each tag)
	// 2 TagCreatedAt indexes (one for each tag)
	// 2 KindTag indexes (one for each tag)
	// 2 KindTagCreatedAt indexes (one for each tag)
	// 2 KindPubkeyTagCreatedAt indexes (one for each tag)
	// 1 Kind index
	// 1 KindPubkey index
	// 1 KindCreatedAt index
	// 1 KindPubkeyCreatedAt index
	// Total: 19 indexes
	expectedIndexCount := 19
	if len(indexes) != expectedIndexCount {
		t.Errorf("Expected %d indexes, got %d", expectedIndexCount, len(indexes))
	}

	// Verify that all indexes are non-nil
	for i, idx := range indexes {
		if idx == nil {
			t.Errorf("Index %d is nil", i)
		}
	}

	// Test with an event that has no tags
	evNoTags := event.New()
	evNoTags.Id = make([]byte, 32)
	evNoTags.Pubkey = make([]byte, 32)
	evNoTags.CreatedAt = timestamp.New(1234567890)
	evNoTags.Kind = kind.New(1)
	evNoTags.Tags = tags.New() // Empty tags
	evNoTags.Content = []byte("Event with no tags")
	evNoTags.Sig = make([]byte, 64)

	// Generate indexes for the event with no tags
	indexesNoTags := GenerateIndexes(evNoTags, serial)

	// For an event with no tags, we expect fewer indexes
	// 1 Event index
	// 1 Id index
	// 1 IdPubkeyCreatedAt index
	// 1 CreatedAt index
	// 1 PubkeyCreatedAt index
	// 1 Kind index
	// 1 KindPubkey index
	// 1 KindCreatedAt index
	// 1 KindPubkeyCreatedAt index
	// Total: 9 indexes
	expectedNoTagsCount := 9
	if len(indexesNoTags) != expectedNoTagsCount {
		t.Errorf("Expected %d indexes for event with no tags, got %d", expectedNoTagsCount, len(indexesNoTags))
	}

	// Verify that all indexes are non-nil
	for i, idx := range indexesNoTags {
		if idx == nil {
			t.Errorf("Index %d is nil for event with no tags", i)
		}
	}

	// Test with an event that has a different kind
	evDifferentKind := event.New()
	evDifferentKind.Id = make([]byte, 32)
	evDifferentKind.Pubkey = make([]byte, 32)
	evDifferentKind.CreatedAt = timestamp.New(1234567890)
	evDifferentKind.Kind = kind.New(30) // Different kind
	evDifferentKind.Tags = tags.New()
	evDifferentKind.Tags.AppendTags(tag.New("e", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"))
	evDifferentKind.Content = []byte("Event with different kind")
	evDifferentKind.Sig = make([]byte, 64)

	// Generate indexes for the event with a different kind
	indexesDifferentKind := GenerateIndexes(evDifferentKind, serial)

	// For an event with a different kind but only one tag, we expect:
	// 1 Event index
	// 1 Id index
	// 1 IdPubkeyCreatedAt index
	// 1 CreatedAt index
	// 1 PubkeyCreatedAt index
	// 1 PubkeyTagCreatedAt index
	// 1 TagCreatedAt index
	// 1 KindTag index
	// 1 KindTagCreatedAt index
	// 1 KindPubkeyTagCreatedAt index
	// 1 Kind index
	// 1 KindPubkey index
	// 1 KindCreatedAt index
	// 1 KindPubkeyCreatedAt index
	// Total: 14 indexes
	expectedDifferentKindCount := 14
	if len(indexesDifferentKind) != expectedDifferentKindCount {
		t.Errorf("Expected %d indexes for event with different kind, got %d", expectedDifferentKindCount, len(indexesDifferentKind))
	}

	// Verify that all indexes are non-nil
	for i, idx := range indexesDifferentKind {
		if idx == nil {
			t.Errorf("Index %d is nil for event with different kind", i)
		}
	}
}
