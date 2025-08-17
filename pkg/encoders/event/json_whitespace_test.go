package event

import (
	"bytes"
	"orly.dev/pkg/utils"
	"testing"

	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
)

// compareEvents compares two events and reports any differences
func compareEvents(t *testing.T, expected, actual *E, context string) {
	if !utils.FastEqual(expected.ID, actual.ID) {
		t.Errorf(
			"%s: ID mismatch: expected %s, got %s", context,
			hex.Enc(expected.ID), hex.Enc(actual.ID),
		)
	}
	if !utils.FastEqual(expected.Pubkey, actual.Pubkey) {
		t.Errorf(
			"%s: Pubkey mismatch: expected %s, got %s", context,
			hex.Enc(expected.Pubkey), hex.Enc(actual.Pubkey),
		)
	}
	if expected.CreatedAt.I64() != actual.CreatedAt.I64() {
		t.Errorf(
			"%s: CreatedAt mismatch: expected %d, got %d", context,
			expected.CreatedAt.I64(), actual.CreatedAt.I64(),
		)
	}
	if expected.Kind.K != actual.Kind.K {
		t.Errorf(
			"%s: Kind mismatch: expected %d, got %d", context, expected.Kind.K,
			actual.Kind.K,
		)
	}
	if !utils.FastEqual(expected.Content, actual.Content) {
		t.Errorf(
			"%s: Content mismatch: expected %s, got %s", context,
			expected.Content, actual.Content,
		)
	}
	if !utils.FastEqual(expected.Sig, actual.Sig) {
		t.Errorf(
			"%s: Sig mismatch: expected %s, got %s", context,
			hex.Enc(expected.Sig), hex.Enc(actual.Sig),
		)
	}
}

func TestMarshalUnmarshalWithWhitespace(t *testing.T) {
	// Create a sample event with predefined values
	original := &E{
		ID:        bytes.Repeat([]byte{0x01}, 32), // 32 bytes of 0x01
		Pubkey:    bytes.Repeat([]byte{0x02}, 32), // 32 bytes of 0x02
		CreatedAt: timestamp.FromUnix(1609459200), // 2021-01-01 00:00:00 UTC
		Kind:      kind.TextNote,                  // Kind 1 (text note)
		Tags:      tags.New(),                     // Empty tags
		Content:   []byte("This is a test event"), // Simple content
		Sig:       bytes.Repeat([]byte{0x03}, 64), // 64 bytes of 0x03
	}

	// Test 1: Marshal with whitespace and unmarshal
	jsonWithWhitespace := original.MarshalWithWhitespace(nil, true)
	parsed := &E{}
	_, err := parsed.Unmarshal(jsonWithWhitespace)
	if err != nil {
		t.Fatalf("Test 1: Failed to unmarshal JSON with whitespace: %v", err)
	}
	compareEvents(t, original, parsed, "Test 1")

	// Test 2: Manually created JSON with extra whitespace
	jsonWithExtraWhitespace := []byte(`
	{
		"id": "` + hex.Enc(original.ID) + `",
		"pubkey": "` + hex.Enc(original.Pubkey) + `",
		"created_at": 1609459200,
		"kind": 1,
		"tags": [],
		"content": "This is a test event",
		"sig": "` + hex.Enc(original.Sig) + `"
	}
	`)
	parsed2 := &E{}
	_, err = parsed2.Unmarshal(jsonWithExtraWhitespace)
	if err != nil {
		t.Fatalf(
			"Test 2: Failed to unmarshal JSON with extra whitespace: %v", err,
		)
	}
	compareEvents(t, original, parsed2, "Test 2")

	// Test 3: JSON with mixed whitespace (spaces, tabs, newlines)
	jsonWithMixedWhitespace := []byte(`{
	"id"  :  "` + hex.Enc(original.ID) + `",
	  "pubkey":	"` + hex.Enc(original.Pubkey) + `",
 "created_at":	 1609459200 ,
		"kind":1,
  "tags":[],
	"content":"This is a test event",
 "sig":"` + hex.Enc(original.Sig) + `"
}`)
	parsed3 := &E{}
	_, err = parsed3.Unmarshal(jsonWithMixedWhitespace)
	if err != nil {
		t.Fatalf(
			"Test 3: Failed to unmarshal JSON with mixed whitespace: %v", err,
		)
	}
	compareEvents(t, original, parsed3, "Test 3")

	// Test 4: JSON with whitespace in unusual places
	jsonWithUnusualWhitespace := []byte(`

	{ 

		"id" : "` + hex.Enc(original.ID) + `" , 
		"pubkey" : "` + hex.Enc(original.Pubkey) + `" , 
		"created_at" : 1609459200 , 
		"kind" : 1 , 
		"tags" : [ ] , 
		"content" : "This is a test event" , 
		"sig" : "` + hex.Enc(original.Sig) + `" 
	} 

	`)
	parsed4 := &E{}
	_, err = parsed4.Unmarshal(jsonWithUnusualWhitespace)
	if err != nil {
		t.Fatalf(
			"Test 4: Failed to unmarshal JSON with unusual whitespace: %v", err,
		)
	}
	compareEvents(t, original, parsed4, "Test 4")

	// Test 5: Minified JSON (no whitespace)
	minifiedJSON := original.Marshal(nil)
	parsed5 := &E{}
	_, err = parsed5.Unmarshal(minifiedJSON)
	if err != nil {
		t.Fatalf("Test 5: Failed to unmarshal minified JSON: %v", err)
	}
	compareEvents(t, original, parsed5, "Test 5")
}
