package database

import (
	"bytes"
	"testing"

	"github.com/minio/sha256-simd"
	"orly.dev/chk"
	"orly.dev/codecbuf"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/event"
	"orly.dev/kind"
	"orly.dev/tag"
	"orly.dev/tags"
	"orly.dev/timestamp"
)

func TestGetIndexesForEvent(t *testing.T) {
	t.Run("BasicEvent", testBasicEvent)
	t.Run("EventWithTags", testEventWithTags)
	t.Run("ErrorHandling", testErrorHandling)
}

// Helper function to verify that a specific index is included in the generated
// indexes
func verifyIndexIncluded(t *testing.T, idxs [][]byte, expectedIdx *indexes.T) {
	// Marshal the expected index
	buf := codecbuf.Get()
	defer codecbuf.Put(buf)
	err := expectedIdx.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("Failed to marshal expected index: %v", err)
	}

	expectedBytes := buf.Bytes()
	found := false

	for _, idx := range idxs {
		if bytes.Equal(idx, expectedBytes) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected index not found in generated indexes")
		t.Errorf("Expected: %v", expectedBytes)
		t.Errorf("Generated indexes: %d indexes", len(idxs))
	}
}

// Test basic event with minimal fields
func testBasicEvent(t *testing.T) {
	// Create a basic event
	ev := event.New()

	// Set ID
	id := make([]byte, sha256.Size)
	for i := range id {
		id[i] = byte(i)
	}
	ev.Id = id

	// Set Pubkey
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i + 1)
	}
	ev.Pubkey = pubkey

	// Set CreatedAt
	ev.CreatedAt = timestamp.FromUnix(12345)

	// Set Kind
	ev.Kind = kind.New(1) // TextNote kind

	// Set Content
	ev.Content = []byte("Test content")

	// Generate indexes
	serial := uint64(1)
	idxs, err := GetIndexesForEvent(ev, serial)
	if chk.E(err) {
		t.Fatalf("GetIndexesForEvent failed: %v", err)
	}

	// Verify the number of indexes (should be 6 for a basic event without tags)
	if len(idxs) != 6 {
		t.Fatalf("Expected 6 indexes, got %d", len(idxs))
	}

	// Create and verify the expected indexes

	// 1. Id index
	ser := new(types.Uint40)
	err = ser.Set(serial)
	if chk.E(err) {
		t.Fatalf("Failed to create Uint40: %v", err)
	}

	idHash := new(types.IdHash)
	err = idHash.FromId(ev.Id)
	if chk.E(err) {
		t.Fatalf("Failed to create IdHash: %v", err)
	}
	idIndex := indexes.IdEnc(idHash, ser)
	verifyIndexIncluded(t, idxs, idIndex)

	// 2. FullIdPubkey index
	fullID := new(types.Id)
	err = fullID.FromId(ev.Id)
	if chk.E(err) {
		t.Fatalf("Failed to create Id: %v", err)
	}

	pubHash := new(types.PubHash)
	err = pubHash.FromPubkey(ev.Pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}

	createdAt := new(types.Uint64)
	createdAt.Set(uint64(ev.CreatedAt.V))

	idPubkeyIndex := indexes.IdPubkeyEnc(ser, fullID, pubHash, createdAt)
	verifyIndexIncluded(t, idxs, idPubkeyIndex)

	// 3. CreatedAt index
	createdAtIndex := indexes.CreatedAtEnc(createdAt, ser)
	verifyIndexIncluded(t, idxs, createdAtIndex)

	// 4. Pubkey index
	pubkeyIndex := indexes.PubkeyEnc(pubHash, createdAt, ser)
	verifyIndexIncluded(t, idxs, pubkeyIndex)

	// 5. Kind index
	kind := new(types.Uint16)
	kind.Set(uint16(ev.Kind.K))

	kindIndex := indexes.KindEnc(kind, createdAt, ser)
	verifyIndexIncluded(t, idxs, kindIndex)

	// 6. KindPubkey index
	kindPubkeyIndex := indexes.KindPubkeyEnc(kind, pubHash, createdAt, ser)
	verifyIndexIncluded(t, idxs, kindPubkeyIndex)
}

// Test event with tags
func testEventWithTags(t *testing.T) {
	// Create an event with tags
	ev := event.New()

	// Set ID
	id := make([]byte, sha256.Size)
	for i := range id {
		id[i] = byte(i)
	}
	ev.Id = id

	// Set Pubkey
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i + 1)
	}
	ev.Pubkey = pubkey

	// Set CreatedAt
	ev.CreatedAt = timestamp.FromUnix(12345)

	// Set Kind
	ev.Kind = kind.New(1) // TextNote kind

	// Set Content
	ev.Content = []byte("Test content with tags")

	// Add tags
	ev.Tags = tags.New()

	// Add e tag (event reference)
	eTagKey := []byte("e")
	eTagValue := []byte("abcdef1234567890")
	eTag := tag.New(eTagKey, eTagValue)
	ev.Tags = ev.Tags.AppendTags(eTag)

	// Add p tag (pubkey reference)
	pTagKey := []byte("p")
	pTagValue := []byte("0123456789abcdef")
	pTag := tag.New(pTagKey, pTagValue)
	ev.Tags = ev.Tags.AppendTags(pTag)

	// Generate indexes
	serial := uint64(2)
	idxs, err := GetIndexesForEvent(ev, serial)
	if chk.E(err) {
		t.Fatalf("GetIndexesForEvent failed: %v", err)
	}

	// Verify the number of indexes (should be 14 for an event with 2 tags)
	// 6 basic indexes + 4 indexes per tag (TagPubkey, Tag, TagKind, TagKindPubkey)
	if len(idxs) != 14 {
		t.Fatalf("Expected 14 indexes, got %d", len(idxs))
	}

	// Create and verify the basic indexes (same as in testBasicEvent)
	ser := new(types.Uint40)
	err = ser.Set(serial)
	if chk.E(err) {
		t.Fatalf("Failed to create Uint40: %v", err)
	}

	idHash := new(types.IdHash)
	err = idHash.FromId(ev.Id)
	if chk.E(err) {
		t.Fatalf("Failed to create IdHash: %v", err)
	}

	// Verify one of the tag-related indexes (e tag)
	pubHash := new(types.PubHash)
	err = pubHash.FromPubkey(ev.Pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}

	createdAt := new(types.Uint64)
	createdAt.Set(uint64(ev.CreatedAt.V))

	// Create tag key and value for e tag
	eKey := new(types.Letter)
	eKey.Set('e')

	eValueHash := new(types.Ident)
	eValueHash.FromIdent([]byte("abcdef1234567890"))

	// Verify TagPubkey index for e tag
	pubkeyTagIndex := indexes.TagPubkeyEnc(
		pubHash, eKey, eValueHash, createdAt, ser,
	)
	verifyIndexIncluded(t, idxs, pubkeyTagIndex)

	// Verify Tag index for e tag
	tagIndex := indexes.TagEnc(
		eKey, eValueHash, createdAt, ser,
	)
	verifyIndexIncluded(t, idxs, tagIndex)

	// Verify TagKind index for e tag
	kind := new(types.Uint16)
	kind.Set(uint16(ev.Kind.K))

	kindTagIndex := indexes.TagKindEnc(
		kind, eKey, eValueHash, createdAt, ser,
	)
	verifyIndexIncluded(t, idxs, kindTagIndex)

	// Verify TagKindPubkey index for e tag
	kindPubkeyTagIndex := indexes.TagKindPubkeyEnc(
		kind, pubHash, eKey, eValueHash, createdAt, ser,
	)
	verifyIndexIncluded(t, idxs, kindPubkeyTagIndex)
}

// Test error handling
func testErrorHandling(t *testing.T) {
	// Test with invalid serial number (too large for Uint40)
	ev := event.New()

	// Set ID
	id := make([]byte, sha256.Size)
	for i := range id {
		id[i] = byte(i)
	}
	ev.Id = id

	// Set Pubkey
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i + 1)
	}
	ev.Pubkey = pubkey

	// Set CreatedAt
	ev.CreatedAt = timestamp.FromUnix(12345)

	// Set Kind
	ev.Kind = kind.New(1) // TextNote kind

	// Set Content
	ev.Content = []byte("Test content")

	// Use an invalid serial number (too large for Uint40)
	invalidSerial := uint64(1) << 40 // 2^40, which is too large for Uint40

	// Generate indexes
	idxs, err := GetIndexesForEvent(ev, invalidSerial)

	// Verify that an error was returned
	if err == nil {
		t.Fatalf("Expected error for invalid serial number, got nil")
	}

	// Verify that idxs is nil when an error occurs
	if idxs != nil {
		t.Fatalf("Expected nil idxs when error occurs, got %v", idxs)
	}

	// Note: We don't test with nil event as it causes a panic
	// The function doesn't have nil checks, which is a potential improvement
}
