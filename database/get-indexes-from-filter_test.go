package database

import (
	"bytes"
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
	t.Run("PubkeyTag", testPubkeyTagFilter)
	t.Run("PubkeyCreatedAt", testPubkeyCreatedAtFilter)
	t.Run("CreatedAt", testCreatedAtFilter)
	t.Run("PubkeyTagCreatedAt", testPubkeyTagCreatedAtFilter)
	t.Run("Tag", testTagFilter)
	t.Run("TagCreatedAt", testTagCreatedAtFilter)
	t.Run("Kind", testKindFilter)
	t.Run("KindCreatedAt", testKindCreatedAtFilter)
	t.Run("KindPubkey", testKindPubkeyFilter)
	t.Run("KindPubkeyCreatedAt", testKindPubkeyCreatedAtFilter)
	t.Run("KindTag", testKindTagFilter)
	t.Run("KindTagCreatedAt", testKindTagCreatedAtFilter)
	t.Run("KindPubkeyTagCreatedAt", testKindPubkeyTagCreatedAtFilter)
}

// Helper function to verify that the generated index matches the expected index
func verifyIndex(t *testing.T, idxs [][]byte, expectedIdx *indexes.T) {
	if len(idxs) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(idxs))
	}

	// Marshal the expected index
	buf := codecbuf.Get()
	defer codecbuf.Put(buf)
	err := expectedIdx.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("Failed to marshal expected index: %v", err)
	}

	// Compare the generated index with the expected index
	if !bytes.Equal(idxs[0], buf.Bytes()) {
		t.Errorf("Generated index does not match expected index")
		t.Errorf("Generated: %v", idxs[0])
		t.Errorf("Expected: %v", buf.Bytes())
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
	ser := new(types.Uint40)
	expectedIdx := indexes.IdEnc(idHash, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test Pubkey filter
func testPubkeyFilter(t *testing.T) {
	// Create a filter with an Author
	f := filter.New()
	pubkey := make([]byte, 32) // Assuming 32 bytes for pubkey
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	f.Authors = f.Authors.Append(pubkey)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected index
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	ser := new(types.Uint40)
	expectedIdx := indexes.PubkeyEnc(p, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test PubkeyTag filter
func testPubkeyTagFilter(t *testing.T) {
	// Create a filter with an Author and a Tag
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

	// Print the filter
	t.Logf("Filter: Authors=%v, Tags=%v", f.Authors.ToSliceOfBytes(), f.Tags.ToSliceOfTags())
	t.Logf("Tag: Key=%v, Value=%v", tagT.B(0), tagT.B(1))

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Print the generated indexes
	t.Logf("Generated indexes: %v", idxs)

	// Create the expected index
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)
	ser := new(types.Uint40)
	expectedIdx := indexes.PubkeyTagEnc(p, key, valueHash, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test PubkeyCreatedAt filter
func testPubkeyCreatedAtFilter(t *testing.T) {
	// Create a filter with an Author and Since
	f := filter.New()
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

	// Create the expected index
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.PubkeyCreatedAtEnc(p, ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
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

	// Create the expected index
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.CreatedAtEnc(ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test PubkeyTagCreatedAt filter
func testPubkeyTagCreatedAtFilter(t *testing.T) {
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

	// Create the expected index
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.PubkeyTagCreatedAtEnc(p, key, valueHash, ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test Tag filter
func testTagFilter(t *testing.T) {
	// Create a filter with a Tag
	f := filter.New()

	// Create a tag
	tagKey := []byte("e")
	tagValue := []byte("test-value")
	tagT := tag.New(tagKey, tagValue)
	f.Tags = f.Tags.AppendTags(tagT)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected index
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)
	ser := new(types.Uint40)
	expectedIdx := indexes.TagEnc(key, valueHash, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test TagCreatedAt filter
func testTagCreatedAtFilter(t *testing.T) {
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

	// Create the expected index
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.TagCreatedAtEnc(key, valueHash, ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test Kind filter
func testKindFilter(t *testing.T) {
	// Create a filter with a Kind
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected index
	kind := new(types.Uint16)
	kind.Set(1)
	ser := new(types.Uint40)
	expectedIdx := indexes.KindEnc(kind, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test KindCreatedAt filter
func testKindCreatedAtFilter(t *testing.T) {
	// Create a filter with a Kind and Since
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))
	f.Since = timestamp.FromUnix(12345)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected index
	kind := new(types.Uint16)
	kind.Set(1)
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.KindCreatedAtEnc(kind, ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test KindPubkey filter
func testKindPubkeyFilter(t *testing.T) {
	// Create a filter with a Kind and an Author
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	f.Authors = f.Authors.Append(pubkey)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected index
	kind := new(types.Uint16)
	kind.Set(1)
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	ser := new(types.Uint40)
	expectedIdx := indexes.KindPubkeyEnc(kind, p, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test KindPubkeyCreatedAt filter
func testKindPubkeyCreatedAtFilter(t *testing.T) {
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

	// Create the expected index
	kind := new(types.Uint16)
	kind.Set(1)
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.KindPubkeyCreatedAtEnc(kind, p, ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test KindTag filter
func testKindTagFilter(t *testing.T) {
	// Create a filter with a Kind and a Tag
	f := filter.New()
	f.Kinds = kinds.New(kind.New(1))

	// Create a tag
	tagKey := []byte("e")
	tagValue := []byte("test-value")
	tagT := tag.New(tagKey, tagValue)
	f.Tags = f.Tags.AppendTags(tagT)

	// Generate indexes
	idxs, err := GetIndexesFromFilter(f)
	if chk.E(err) {
		t.Fatalf("GetIndexesFromFilter failed: %v", err)
	}

	// Create the expected index
	kind := new(types.Uint16)
	kind.Set(1)
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)
	ser := new(types.Uint40)
	expectedIdx := indexes.KindTagEnc(kind, key, valueHash, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test KindTagCreatedAt filter
func testKindTagCreatedAtFilter(t *testing.T) {
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

	// Create the expected index
	kind := new(types.Uint16)
	kind.Set(1)
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.KindTagCreatedAtEnc(kind, key, valueHash, ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}

// Test KindPubkeyTagCreatedAt filter
func testKindPubkeyTagCreatedAtFilter(t *testing.T) {
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

	// Create the expected index
	kind := new(types.Uint16)
	kind.Set(1)
	p := new(types.PubHash)
	err = p.FromPubkey(pubkey)
	if chk.E(err) {
		t.Fatalf("Failed to create PubHash: %v", err)
	}
	key := new(types.Letter)
	key.Set(tagKey[0])
	valueHash := new(types.Ident)
	valueHash.FromIdent(tagValue)
	ca := new(types.Uint64)
	ca.Set(uint64(f.Since.V))
	ser := new(types.Uint40)
	expectedIdx := indexes.KindPubkeyTagCreatedAtEnc(kind, p, key, valueHash, ca, ser)

	// Verify the generated index
	verifyIndex(t, idxs, expectedIdx)
}
