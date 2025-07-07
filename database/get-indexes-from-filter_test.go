package database

import (
	"bytes"
	"math"
	"testing"

	"github.com/minio/sha256-simd"
	"orly.dev/chk"
	"orly.dev/codecbuf"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
	"orly.dev/kind"
	"orly.dev/kinds"
	"orly.dev/tag"
	"orly.dev/timestamp"
)

// TestGetIndexesFromFilter tests the GetIndexesFromFilter function
func TestGetIndexesFromFilter(t *testing.T) {
	// Test cases for each filter type
	t.Run("Id", testIdFilter)
	t.Run("Pubkey", testPubkeyFilter)
	t.Run("CreatedAt", testCreatedAtFilter)
	t.Run("CreatedAtUntil", testCreatedAtUntilFilter)
	t.Run("PubkeyTag", testPubkeyTagFilter)
	t.Run("Tag", testTagFilter)
	t.Run("Kind", testKindFilter)
	t.Run("KindPubkey", testKindPubkeyFilter)
	t.Run("KindTag", testKindTagFilter)
	t.Run("KindPubkeyTag", testKindPubkeyTagFilter)
}

// Helper function to verify that the generated index matches the expected indexes
func verifyIndex(
	t *testing.T, idxs []Range, expectedStartIdx, expectedEndIdx *indexes.T,
) {
	if len(idxs) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(idxs))
	}

	// Marshal the expected start index
	startBuf := codecbuf.Get()
	defer codecbuf.Put(startBuf)
	err := expectedStartIdx.MarshalWrite(startBuf)
	if chk.E(err) {
		t.Fatalf("Failed to marshal expected start index: %v", err)
	}

	// Compare the generated start index with the expected start index
	if !bytes.Equal(idxs[0].Start, startBuf.Bytes()) {
		t.Errorf("Generated start index does not match expected start index")
		t.Errorf("Generated: %v", idxs[0].Start)
		t.Errorf("Expected: %v", startBuf.Bytes())
	}

	// If expectedEndIdx is nil, use expectedStartIdx
	endIdx := expectedEndIdx
	if endIdx == nil {
		endIdx = expectedStartIdx
	}

	// Marshal the expected end index
	endBuf := codecbuf.Get()
	defer codecbuf.Put(endBuf)
	err = endIdx.MarshalWrite(endBuf)
	if chk.E(err) {
		t.Fatalf("Failed to marshal expected End index: %v", err)
	}

	// Compare the generated end index with the expected end index
	if !bytes.Equal(idxs[0].End, endBuf.Bytes()) {
		t.Errorf("Generated End index does not match expected End index")
		t.Errorf("Generated: %v", idxs[0].End)
		t.Errorf("Expected: %v", endBuf.Bytes())
	}
}

// Test Id filter
func testIdFilter(t *testing.T) {
	// Create a filter with an Id
	f := filter.New()
	id := make([]byte, sha256.Size)
	for i := range id {
		id[i] = byte(i)
	}
	f.Ids = f.Ids.Append(id)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected index
	idHash := new(types.IdHash)
	err = idHash.FromId(id)
	if chk.E(err) {
		t.Fatalf("Failed to create IdHash: %v", err)
	}
	expectedIdx := indexes.IdEnc(idHash, nil)

	// Verify the generated index
	// For Id filter, both start and end indexes are the same
	verifyIndex(t, idxs, expectedIdx, expectedIdx)
}

// Test Pubkey filter
func testPubkeyFilter(t *testing.T) {
	// Create a filter with an Author, Since, and Until
	f := filter.New()
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	f.Authors = f.Authors.Append(pubkey)
	f.Since = timestamp.FromUnix(12345)
	f.Until = timestamp.FromUnix(67890) // Added Until field

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected indexes
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}

	// Start index uses Since
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.PubkeyEnc(p, caStart, nil)

	// End index uses Until
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(f.Until.V))
	expectedEndIdx := indexes.PubkeyEnc(p, caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test CreatedAt filter
func testCreatedAtFilter(t *testing.T) {
	// Create a filter with Since
	f := filter.New()
	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected start index (using Since)
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.CreatedAtEnc(caStart, nil)

	// Create the expected end index (using math.MaxInt64 since Until is not specified)
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(math.MaxInt64))
	expectedEndIdx := indexes.CreatedAtEnc(caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test CreatedAt filter with Until
func testCreatedAtUntilFilter(t *testing.T) {
	// Create a filter with Until
	f := filter.New()
	f.Until = timestamp.FromUnix(67890)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected start index (using 0 since Since is not specified)
	caStart := new(types.Uint64)
	caStart.Set(uint64(0))
	expectedStartIdx := indexes.CreatedAtEnc(caStart, nil)

	// Create the expected end index (using Until)
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(f.Until.V))
	expectedEndIdx := indexes.CreatedAtEnc(caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test PubkeyTag filter
func testPubkeyTagFilter(t *testing.T) {
	// Create a filter with an Author, a Tag, and Since
	f := filter.New()
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	f.Authors = f.Authors.Append(pubkey)

	// Create a tag
	tagKey := []byte("e")
	tagValue := []byte("test-value")
	tagT := tag.New(tagKey, tagValue)
	f.Tags = f.Tags.AppendTags(tagT)

	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected indexes
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)

	// Start index uses Since
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.PubkeyTagEnc(p, key, valueHash, caStart, nil)

	// End index uses math.MaxInt64 since Until is not specified
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(math.MaxInt64))
	expectedEndIdx := indexes.PubkeyTagEnc(p, key, valueHash, caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test Tag filter
func testTagFilter(t *testing.T) {
	// Create a filter with a Tag and Since
	f := filter.New()

	// Create a tag
	tagKey := []byte("e")
	tagValue := []byte("test-value")
	tagT := tag.New(tagKey, tagValue)
	f.Tags = f.Tags.AppendTags(tagT)

	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected indexes
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)

	// Start index uses Since
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.TagEnc(key, valueHash, caStart, nil)

	// End index uses math.MaxInt64 since Until is not specified
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(math.MaxInt64))
	expectedEndIdx := indexes.TagEnc(key, valueHash, caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test Kind filter
func testKindFilter(t *testing.T) {
	// Create a filter with a Kind and Since
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))
	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected indexes
	k := new(types.Uint16)
	k.Set(1)

	// Start index uses Since
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.KindEnc(k, caStart, nil)

	// End index uses math.MaxInt64 since Until is not specified
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(math.MaxInt64))
	expectedEndIdx := indexes.KindEnc(k, caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test KindPubkey filter
func testKindPubkeyFilter(t *testing.T) {
	// Create a filter with a Kind, an Author, and Since
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	f.Authors = f.Authors.Append(pubkey)
	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected indexes
	k := new(types.Uint16)
	k.Set(1)
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}

	// Start index uses Since
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.KindPubkeyEnc(k, p, caStart, nil)

	// End index uses math.MaxInt64 since Until is not specified
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(math.MaxInt64))
	expectedEndIdx := indexes.KindPubkeyEnc(k, p, caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test KindTag filter
func testKindTagFilter(t *testing.T) {
	// Create a filter with a Kind, a Tag, and Since
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))

	// Create a tag
	tagKey := []byte("e")
	tagValue := []byte("test-value")
	tagT := tag.New(tagKey, tagValue)
	f.Tags = f.Tags.AppendTags(tagT)

	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected indexes
	k := new(types.Uint16)
	k.Set(1)
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)

	// Start index uses Since
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.KindTagEnc(k, key, valueHash, caStart, nil)

	// End index uses math.MaxInt64 since Until is not specified
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(math.MaxInt64))
	expectedEndIdx := indexes.KindTagEnc(k, key, valueHash, caEnd, nil)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}

// Test KindPubkeyTag filter
func testKindPubkeyTagFilter(t *testing.T) {
	// Create a filter with a Kind, an Author, a Tag, and Since
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	f.Authors = f.Authors.Append(pubkey)

	// Create a tag
	tagKey := []byte("e")
	tagValue := []byte("test-value")
	tagT := tag.New(tagKey, tagValue)
	f.Tags = f.Tags.AppendTags(tagT)

	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected indexes
	k := new(types.Uint16)
	k.Set(1)
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)

	// Start index uses Since
	caStart := new(types.Uint64)
	caStart.Set(uint64(f.Since.V))
	expectedStartIdx := indexes.KindPubkeyTagEnc(
		k, p, key, valueHash, caStart, nil,
	)

	// End index uses math.MaxInt64 since Until is not specified
	caEnd := new(types.Uint64)
	caEnd.Set(uint64(math.MaxInt64))
	expectedEndIdx := indexes.KindPubkeyTagEnc(
		k, p, key, valueHash, caEnd, nil,
	)

	// Verify the generated index
	verifyIndex(t, idxs, expectedStartIdx, expectedEndIdx)
}
