package idhash

import (
	"bytes"
	"encoding/base64"
	"not.realy.lol/codecbuf"
	"testing"

	"github.com/minio/sha256-simd"
	"not.realy.lol/chk"
	"not.realy.lol/hex"
)

func TestNew(t *testing.T) {
	i := New()
	if i == nil {
		t.Fatal("New() returned nil")
	}
	if len(i.val) != Len {
		t.Errorf(
			"New() created a T with val length %d, want %d", len(i.val), Len,
		)
	}
}

func TestFromId(t *testing.T) {
	// Create a valid ID (32 bytes)
	validId := make([]byte, sha256.Size)
	for i := 0; i < sha256.Size; i++ {
		validId[i] = byte(i)
	}

	// Create an invalid ID (wrong size)
	invalidId := make([]byte, sha256.Size-1)

	// Test with valid ID
	i := New()
	err := i.FromId(validId)
	if chk.E(err) {
		t.Fatalf("FromId failed with valid ID: %v", err)
	}

	// Calculate the expected hash
	idh := sha256.Sum256(validId)
	expected := idh[:Len]

	// Verify the hash was set correctly
	if !bytes.Equal(i.Bytes(), expected) {
		t.Errorf(
			"FromId did not set the hash correctly: got %v, want %v", i.Bytes(),
			expected,
		)
	}

	// Test with invalid ID
	i = New()
	err = i.FromId(invalidId)
	if err == nil {
		t.Errorf("FromId should have failed with invalid ID size")
	}
}

func TestFromIdBase64(t *testing.T) {
	// Create a valid ID (32 bytes)
	validId := make([]byte, sha256.Size)
	for i := 0; i < sha256.Size; i++ {
		validId[i] = byte(i)
	}

	// Encode the ID as base64
	validIdBase64 := base64.RawURLEncoding.EncodeToString(validId)

	// Test with valid base64 ID
	i := New()
	err := i.FromIdBase64(validIdBase64)
	if chk.E(err) {
		t.Fatalf("FromIdBase64 failed with valid ID: %v", err)
	}

	// Calculate the expected hash
	idh := sha256.Sum256(validId)
	expected := idh[:Len]

	// Verify the hash was set correctly
	if !bytes.Equal(i.Bytes(), expected) {
		t.Errorf(
			"FromIdBase64 did not set the hash correctly: got %v, want %v",
			i.Bytes(), expected,
		)
	}

	// Test with invalid base64 ID
	i = New()
	err = i.FromIdBase64("invalid-base64")
	if err == nil {
		t.Errorf("FromIdBase64 should have failed with invalid base64")
	}
}

func TestFromIdHex(t *testing.T) {
	// Create a valid ID (32 bytes)
	validId := make([]byte, sha256.Size)
	for i := 0; i < sha256.Size; i++ {
		validId[i] = byte(i)
	}

	// Encode the ID as hex
	validIdHex := hex.Enc(validId)

	// Test with valid hex ID
	i := New()
	err := i.FromIdHex(validIdHex)
	if chk.E(err) {
		t.Fatalf("FromIdHex failed with valid ID: %v", err)
	}

	// Calculate the expected hash
	idh := sha256.Sum256(validId)
	expected := idh[:Len]

	// Verify the hash was set correctly
	if !bytes.Equal(i.Bytes(), expected) {
		t.Errorf(
			"FromIdHex did not set the hash correctly: got %v, want %v",
			i.Bytes(), expected,
		)
	}

	// Test with invalid hex ID (wrong size)
	i = New()
	err = i.FromIdHex(validIdHex[:len(validIdHex)-2])
	if err == nil {
		t.Errorf("FromIdHex should have failed with invalid ID size")
	}

	// Test with invalid hex ID (not hex)
	i = New()
	err = i.FromIdHex("invalid-hex")
	if err == nil {
		t.Errorf("FromIdHex should have failed with invalid hex")
	}
}

func TestMarshalWriteUnmarshalRead(t *testing.T) {
	// Create a T with a known value
	i1 := New()
	validId := make([]byte, sha256.Size)
	for i := 0; i < sha256.Size; i++ {
		validId[i] = byte(i)
	}
	err := i1.FromId(validId)
	if chk.E(err) {
		t.Fatalf("FromId failed: %v", err)
	}

	// Test MarshalWrite
	buf := codecbuf.Get()
	err = i1.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Verify the written bytes
	if !bytes.Equal(buf.Bytes(), i1.Bytes()) {
		t.Errorf("MarshalWrite wrote %v, want %v", buf.Bytes(), i1.Bytes())
	}

	// Test UnmarshalRead
	i2 := New()
	err = i2.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the read value
	if !bytes.Equal(i2.Bytes(), i1.Bytes()) {
		t.Errorf("UnmarshalRead read %v, want %v", i2.Bytes(), i1.Bytes())
	}
}

func TestUnmarshalReadWithEmptyVal(t *testing.T) {
	// Create a T with an empty val
	i := &T{val: []byte{}}

	// Create some test data
	testData := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	// Test UnmarshalRead
	err := i.UnmarshalRead(bytes.NewBuffer(testData))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the read value
	if !bytes.Equal(i.Bytes(), testData) {
		t.Errorf("UnmarshalRead read %v, want %v", i.Bytes(), testData)
	}
}

func TestUnmarshalReadWithSmallerVal(t *testing.T) {
	// Create a T with a val smaller than Len
	i := &T{val: make([]byte, Len-1)}

	// Create some test data
	testData := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	// Test UnmarshalRead
	err := i.UnmarshalRead(bytes.NewBuffer(testData))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the read value
	if !bytes.Equal(i.Bytes(), testData) {
		t.Errorf("UnmarshalRead read %v, want %v", i.Bytes(), testData)
	}
}

func TestUnmarshalReadWithLargerVal(t *testing.T) {
	// Create a T with a val larger than Len
	i := &T{val: make([]byte, Len+1)}

	// Create some test data
	testData := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	// Test UnmarshalRead
	err := i.UnmarshalRead(bytes.NewBuffer(testData))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the read value
	if !bytes.Equal(i.Bytes(), testData) {
		t.Errorf("UnmarshalRead read %v, want %v", i.Bytes(), testData)
	}
}
