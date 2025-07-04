package pubhash

import (
	"bytes"
	"testing"

	"github.com/minio/sha256-simd"
	"not.realy.lol/chk"
	"not.realy.lol/ec/schnorr"
	"not.realy.lol/hex"
)

func TestFromPubkey(t *testing.T) {
	// Create a valid pubkey (32 bytes)
	validPubkey := make([]byte, schnorr.PubKeyBytesLen)
	for i := 0; i < schnorr.PubKeyBytesLen; i++ {
		validPubkey[i] = byte(i)
	}

	// Create an invalid pubkey (wrong size)
	invalidPubkey := make([]byte, schnorr.PubKeyBytesLen-1)

	// Test with valid pubkey
	ph := &T{}
	err := ph.FromPubkey(validPubkey)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed with valid pubkey: %v", err)
	}

	// Calculate the expected hash
	pkh := sha256.Sum256(validPubkey)
	expected := pkh[:Len]

	// Verify the hash was set correctly
	if !bytes.Equal(ph.Bytes(), expected) {
		t.Errorf("FromPubkey did not set the hash correctly: got %v, want %v", ph.Bytes(), expected)
	}

	// Test with invalid pubkey
	ph = &T{}
	err = ph.FromPubkey(invalidPubkey)
	if err == nil {
		t.Errorf("FromPubkey should have failed with invalid pubkey size")
	}
}

func TestFromPubkeyHex(t *testing.T) {
	// Create a valid pubkey (32 bytes)
	validPubkey := make([]byte, schnorr.PubKeyBytesLen)
	for i := 0; i < schnorr.PubKeyBytesLen; i++ {
		validPubkey[i] = byte(i)
	}

	// Encode the pubkey as hex
	validPubkeyHex := hex.Enc(validPubkey)

	// Test with valid hex pubkey
	ph := &T{}
	err := ph.FromPubkeyHex(validPubkeyHex)
	if chk.E(err) {
		t.Fatalf("FromPubkeyHex failed with valid pubkey: %v", err)
	}

	// Calculate the expected hash
	pkh := sha256.Sum256(validPubkey)
	expected := pkh[:Len]

	// Verify the hash was set correctly
	if !bytes.Equal(ph.Bytes(), expected) {
		t.Errorf("FromPubkeyHex did not set the hash correctly: got %v, want %v", ph.Bytes(), expected)
	}

	// Test with invalid hex pubkey (wrong size)
	ph = &T{}
	err = ph.FromPubkeyHex(validPubkeyHex[:len(validPubkeyHex)-2])
	if err == nil {
		t.Errorf("FromPubkeyHex should have failed with invalid pubkey size")
	}

	// Test with invalid hex pubkey (not hex)
	ph = &T{}
	err = ph.FromPubkeyHex("invalid-hex")
	if err == nil {
		t.Errorf("FromPubkeyHex should have failed with invalid hex")
	}
}

func TestMarshalWriteUnmarshalRead(t *testing.T) {
	// Create a T with a known value
	ph1 := &T{}
	validPubkey := make([]byte, schnorr.PubKeyBytesLen)
	for i := 0; i < schnorr.PubKeyBytesLen; i++ {
		validPubkey[i] = byte(i)
	}
	err := ph1.FromPubkey(validPubkey)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}

	// Test MarshalWrite
	buf := new(bytes.Buffer)
	err = ph1.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Verify the written bytes
	if !bytes.Equal(buf.Bytes(), ph1.Bytes()) {
		t.Errorf("MarshalWrite wrote %v, want %v", buf.Bytes(), ph1.Bytes())
	}

	// Test UnmarshalRead
	ph2 := &T{}
	err = ph2.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the read value
	if !bytes.Equal(ph2.Bytes(), ph1.Bytes()) {
		t.Errorf("UnmarshalRead read %v, want %v", ph2.Bytes(), ph1.Bytes())
	}
}

func TestUnmarshalReadWithCorruptedData(t *testing.T) {
	// Create a T with a known value
	ph1 := &T{}
	validPubkey1 := make([]byte, schnorr.PubKeyBytesLen)
	for i := 0; i < schnorr.PubKeyBytesLen; i++ {
		validPubkey1[i] = byte(i)
	}
	err := ph1.FromPubkey(validPubkey1)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}

	// Create a second T with a different value
	ph2 := &T{}
	validPubkey2 := make([]byte, schnorr.PubKeyBytesLen)
	for i := 0; i < schnorr.PubKeyBytesLen; i++ {
		validPubkey2[i] = byte(schnorr.PubKeyBytesLen - i - 1)
	}
	err = ph2.FromPubkey(validPubkey2)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}

	// Test UnmarshalRead with corrupted data (less than Len bytes)
	corruptedData := make([]byte, Len/2)
	ph2.UnmarshalRead(bytes.NewBuffer(corruptedData))

	// The UnmarshalRead method should not have copied the original data to itself
	// before reading, so the value should be partially overwritten
	if bytes.Equal(ph2.Bytes(), ph1.Bytes()) {
		t.Errorf("UnmarshalRead did not modify the value as expected")
	}
}