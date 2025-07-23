package event

import (
	"bytes"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	text2 "orly.dev/pkg/encoders/text"
	"orly.dev/pkg/encoders/timestamp"
	"testing"
)

// compareTags compares two tags and reports any differences
func compareTags(t *testing.T, expected, actual *tags.T, context string) {
	if expected == nil && actual == nil {
		return
	}

	if expected == nil || actual == nil {
		t.Errorf("%s: One of the tags is nil", context)
		return
	}

	expectedSlice := expected.ToStringsSlice()
	actualSlice := actual.ToStringsSlice()

	if len(expectedSlice) != len(actualSlice) {
		t.Errorf(
			"%s: Tags length mismatch: expected %d, got %d", context,
			len(expectedSlice), len(actualSlice),
		)
		return
	}

	for i, expectedTag := range expectedSlice {
		actualTag := actualSlice[i]

		if len(expectedTag) != len(actualTag) {
			t.Errorf(
				"%s: Tag[%d] length mismatch: expected %d, got %d", context, i,
				len(expectedTag), len(actualTag),
			)
			continue
		}

		for j, expectedElem := range expectedTag {
			if expectedElem != actualTag[j] {
				t.Errorf(
					"%s: Tag[%d][%d] mismatch: expected '%s', got '%s'",
					context, i, j, expectedElem, actualTag[j],
				)
			}
		}
	}
}

// TestUnmarshalEscapedJSONInTags tests that the Unmarshal function correctly handles
// tags with fields containing escaped JSON that has been escaped using NostrEscape.
func TestUnmarshalEscapedJSONInTags(t *testing.T) {
	// Test 1: Tag with a field containing escaped JSON
	t.Run(
		"SimpleEscapedJSON", func(t *testing.T) {
			// Create a tag with a field containing JSON that needs escaping
			jsonContent := `{"key":"value","nested":{"array":[1,2,3]}}`

			// Create the event with the tag containing JSON
			originalEvent := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("Event with JSON in tag"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add a tag with JSON content
			jsonTag := tag.New("j", jsonContent)
			originalEvent.Tags.AppendTags(jsonTag)

			// Marshal the event
			marshaled := originalEvent.Marshal(nil)

			// Unmarshal back into a new event
			unmarshaledEvent := &E{}
			_, err := unmarshaledEvent.Unmarshal(marshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal event with JSON in tag: %v", err)
			}

			// Verify the tag was correctly unmarshaled
			if unmarshaledEvent.Tags.Len() != 1 {
				t.Fatalf("Expected 1 tag, got %d", unmarshaledEvent.Tags.Len())
			}

			unmarshaledTag := unmarshaledEvent.Tags.GetTagElement(0)
			if unmarshaledTag.Len() != 2 {
				t.Fatalf(
					"Expected tag with 2 elements, got %d", unmarshaledTag.Len(),
				)
			}

			if string(unmarshaledTag.B(0)) != "j" {
				t.Errorf("Expected tag key 'j', got '%s'", unmarshaledTag.B(0))
			}

			if string(unmarshaledTag.B(1)) != jsonContent {
				t.Errorf(
					"Expected tag value '%s', got '%s'", jsonContent,
					unmarshaledTag.B(1),
				)
			}
		},
	)

	// Test 2: Tag with a field containing escaped JSON with special characters
	t.Run(
		"EscapedJSONWithSpecialChars", func(t *testing.T) {
			// JSON with characters that need escaping: quotes, backslashes, control chars
			jsonContent := `{"text":"This has \"quotes\" and \\ backslashes","newlines":"\n\r\t"}`

			// Create the event with the tag containing JSON with special chars
			originalEvent := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("Event with JSON containing special chars in tag"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add a tag with JSON content containing special chars
			jsonTag := tag.New("j", jsonContent)
			originalEvent.Tags.AppendTags(jsonTag)

			// Marshal the event
			marshaled := originalEvent.Marshal(nil)

			// Unmarshal back into a new event
			unmarshaledEvent := &E{}
			_, err := unmarshaledEvent.Unmarshal(marshaled)
			if err != nil {
				t.Fatalf(
					"Failed to unmarshal event with JSON containing special chars: %v",
					err,
				)
			}

			// Verify the tag was correctly unmarshaled
			unmarshaledTag := unmarshaledEvent.Tags.GetTagElement(0)
			if string(unmarshaledTag.B(1)) != jsonContent {
				t.Errorf(
					"Expected tag value '%s', got '%s'", jsonContent,
					unmarshaledTag.B(1),
				)
			}
		},
	)

	// Test 3: Tag with nested JSON that contains already escaped content
	t.Run(
		"NestedEscapedJSON", func(t *testing.T) {
			// JSON with already escaped content
			jsonContent := `{"escaped":"This JSON contains \\\"already escaped\\\" content"}`

			// Create the event with the tag containing nested escaped JSON
			originalEvent := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("Event with nested escaped JSON in tag"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add a tag with nested escaped JSON content
			jsonTag := tag.New("j", jsonContent)
			originalEvent.Tags.AppendTags(jsonTag)

			// Marshal the event
			marshaled := originalEvent.Marshal(nil)

			// Unmarshal back into a new event
			unmarshaledEvent := &E{}
			_, err := unmarshaledEvent.Unmarshal(marshaled)
			if err != nil {
				t.Fatalf(
					"Failed to unmarshal event with nested escaped JSON: %v",
					err,
				)
			}

			// Verify the tag was correctly unmarshaled
			unmarshaledTag := unmarshaledEvent.Tags.GetTagElement(0)
			if string(unmarshaledTag.B(1)) != jsonContent {
				t.Errorf(
					"Expected tag value '%s', got '%s'", jsonContent,
					unmarshaledTag.B(1),
				)
			}
		},
	)

	// Test 4: Tag with JSON that has been explicitly escaped using NostrEscape
	t.Run(
		"ExplicitlyEscapedJSON", func(t *testing.T) {
			// Original JSON with characters that need escaping
			originalJSON := []byte(`{"key":"value with "quotes"","nested":{"array":[1,2,3],"special":"\n\r\t"}}`)

			// Explicitly escape the JSON using NostrEscape
			escapedJSON := make([]byte, 0, len(originalJSON)*2)
			escapedJSON = text2.NostrEscape(escapedJSON, originalJSON)

			// Create the event with the tag containing explicitly escaped JSON
			originalEvent := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("Event with explicitly escaped JSON in tag"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add a tag with the explicitly escaped JSON content
			jsonTag := tag.New("j", string(escapedJSON))
			originalEvent.Tags.AppendTags(jsonTag)

			// Marshal the event
			marshaled := originalEvent.Marshal(nil)

			// Unmarshal back into a new event
			unmarshaledEvent := &E{}
			_, err := unmarshaledEvent.Unmarshal(marshaled)
			if err != nil {
				t.Fatalf(
					"Failed to unmarshal event with explicitly escaped JSON: %v",
					err,
				)
			}

			// Verify the tag was correctly unmarshaled
			unmarshaledTag := unmarshaledEvent.Tags.GetTagElement(0)
			if string(unmarshaledTag.B(1)) != string(escapedJSON) {
				t.Errorf(
					"Expected tag value '%s', got '%s'", string(escapedJSON),
					unmarshaledTag.B(1),
				)
			}

			// Unescape the unmarshaled JSON to verify it matches the original
			unescapedJSON := make([]byte, len(unmarshaledTag.B(1)))
			copy(unescapedJSON, unmarshaledTag.B(1))
			unescapedJSON = text2.NostrUnescape(unescapedJSON)

			if string(unescapedJSON) != string(originalJSON) {
				t.Errorf(
					"Unescaped JSON doesn't match original. Expected '%s', got '%s'",
					string(originalJSON), string(unescapedJSON),
				)
			}
		},
	)
}

func TestUnmarshalTags(t *testing.T) {
	// Test 1: Simple event with empty tags
	t.Run(
		"EmptyTags", func(t *testing.T) {
			jsonWithEmptyTags := []byte(`{"id":"0101010101010101010101010101010101010101010101010101010101010101","pubkey":"0202020202020202020202020202020202020202020202020202020202020202","created_at":1609459200,"kind":1,"tags":[],"content":"This is a test event","sig":"03030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303"}`)

			expected := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("This is a test event"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			actual := &E{}
			_, err := actual.Unmarshal(jsonWithEmptyTags)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON with empty tags: %v", err)
			}

			compareTags(t, expected.Tags, actual.Tags, "EmptyTags")
		},
	)

	// Test 2: Event with simple tags
	t.Run(
		"SimpleTags", func(t *testing.T) {
			jsonWithSimpleTags := []byte(`{"id":"0101010101010101010101010101010101010101010101010101010101010101","pubkey":"0202020202020202020202020202020202020202020202020202020202020202","created_at":1609459200,"kind":1,"tags":[["e","1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"],["p","abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"]],"content":"This is a test event","sig":"03030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303"}`)

			expected := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("This is a test event"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add tags
			eTag := tag.New(
				"e",
				"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			)
			pTag := tag.New(
				"p",
				"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			)
			expected.Tags.AppendTags(eTag, pTag)

			actual := &E{}
			_, err := actual.Unmarshal(jsonWithSimpleTags)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON with simple tags: %v", err)
			}

			compareTags(t, expected.Tags, actual.Tags, "SimpleTags")
		},
	)

	// Test 3: Event with complex tags (more elements per tag)
	t.Run(
		"ComplexTags", func(t *testing.T) {
			jsonWithComplexTags := []byte(`{"id":"0101010101010101010101010101010101010101010101010101010101010101","pubkey":"0202020202020202020202020202020202020202020202020202020202020202","created_at":1609459200,"kind":1,"tags":[["e","1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef","wss://relay.example.com","root"],["p","abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890","wss://relay.example.com"],["t","hashtag","topic"]],"content":"This is a test event","sig":"03030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303"}`)

			expected := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("This is a test event"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add tags
			eTag := tag.New(
				"e",
				"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"wss://relay.example.com", "root",
			)
			pTag := tag.New(
				"p",
				"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				"wss://relay.example.com",
			)
			tTag := tag.New("t", "hashtag", "topic")
			expected.Tags.AppendTags(eTag, pTag, tTag)

			actual := &E{}
			_, err := actual.Unmarshal(jsonWithComplexTags)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON with complex tags: %v", err)
			}

			compareTags(t, expected.Tags, actual.Tags, "ComplexTags")
		},
	)

	// Test 4: Test using the Unmarshal function (not the method)
	t.Run(
		"UnmarshalFunction", func(t *testing.T) {
			jsonWithTags := []byte(`{
			"id": "0101010101010101010101010101010101010101010101010101010101010101",
			"pubkey": "0202020202020202020202020202020202020202020202020202020202020202",
			"created_at": 1609459200,
			"kind": 1,
			"tags": [["e", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"], ["p", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"]],
			"content": "This is a test event",
			"sig": "03030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303"
		}`)

			expected := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("This is a test event"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add tags
			eTag := tag.New(
				"e",
				"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			)
			pTag := tag.New(
				"p",
				"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			)
			expected.Tags.AppendTags(eTag, pTag)

			actual := &E{}
			_, err := Unmarshal(actual, jsonWithTags)
			if err != nil {
				t.Fatalf(
					"Failed to unmarshal JSON with tags using Unmarshal function: %v",
					err,
				)
			}

			compareTags(t, expected.Tags, actual.Tags, "UnmarshalFunction")
		},
	)

	// Test 5: Event with nested empty tags
	t.Run(
		"NestedEmptyTags", func(t *testing.T) {
			jsonWithNestedEmptyTags := []byte(`{
			"id": "0101010101010101010101010101010101010101010101010101010101010101",
			"pubkey": "0202020202020202020202020202020202020202020202020202020202020202",
			"created_at": 1609459200,
			"kind": 1,
			"tags": [[], ["e"], ["p", ""]],
			"content": "This is a test event",
			"sig": "03030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303"
		}`)

			expected := &E{
				ID:        bytes.Repeat([]byte{0x01}, 32),
				Pubkey:    bytes.Repeat([]byte{0x02}, 32),
				CreatedAt: timestamp.FromUnix(1609459200),
				Kind:      kind.TextNote,
				Tags:      tags.New(),
				Content:   []byte("This is a test event"),
				Sig:       bytes.Repeat([]byte{0x03}, 64),
			}

			// Add tags
			emptyTag := tag.New[string]()
			eTag := tag.New("e")
			pTag := tag.New("p", "")
			expected.Tags.AppendTags(emptyTag, eTag, pTag)

			actual := &E{}
			_, err := actual.Unmarshal(jsonWithNestedEmptyTags)
			if err != nil {
				t.Fatalf(
					"Failed to unmarshal JSON with nested empty tags: %v", err,
				)
			}

			compareTags(t, expected.Tags, actual.Tags, "NestedEmptyTags")
		},
	)
}
