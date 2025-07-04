package letter

import (
	"bytes"
	"not.realy.lol/codecbuf"
	"testing"

	"not.realy.lol/chk"
)

func TestNew(t *testing.T) {
	// Test with a valid letter
	l := New('A')
	if l == nil {
		t.Fatal("New() returned nil")
	}
	if l.Letter() != 'A' {
		t.Errorf(
			"New('A') created a T with letter %c, want %c", l.Letter(), 'A',
		)
	}
}

func TestSet(t *testing.T) {
	// Create a T with a known value
	l := New('A')

	// Test Set
	l.Set('B')
	if l.Letter() != 'B' {
		t.Errorf(
			"Set('B') did not set the letter correctly: got %c, want %c",
			l.Letter(), 'B',
		)
	}
}

func TestLetter(t *testing.T) {
	// Create a T with a known value
	l := New('A')

	// Test Letter
	if l.Letter() != 'A' {
		t.Errorf("Letter() returned %c, want %c", l.Letter(), 'A')
	}
}

func TestMarshalWriteUnmarshalRead(t *testing.T) {
	// Create a T with a known value
	l1 := New('A')

	// Test MarshalWrite
	buf := codecbuf.Get()
	err := l1.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Verify the written bytes
	if buf.Len() != 1 || buf.Bytes()[0] != 'A' {
		t.Errorf("MarshalWrite wrote %v, want [%d]", buf.Bytes(), 'A')
	}

	// Test UnmarshalRead
	l2 := New('B') // Start with a different value
	err = l2.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the read value
	if l2.Letter() != 'A' {
		t.Errorf("UnmarshalRead read %c, want %c", l2.Letter(), 'A')
	}
}

func TestUnmarshalReadWithEmptyReader(t *testing.T) {
	// Create a T with a known value
	l := New('A')

	// Test UnmarshalRead with an empty reader
	err := l.UnmarshalRead(bytes.NewBuffer([]byte{}))
	if err == nil {
		t.Errorf("UnmarshalRead should have failed with an empty reader")
	}
}

func TestEdgeCases(t *testing.T) {
	// Test with minimum value (0)
	l1 := New(0)
	if l1.Letter() != 0 {
		t.Errorf("New(0) created a T with letter %d, want %d", l1.Letter(), 0)
	}

	// Test with maximum value (255)
	l2 := New(255)
	if l2.Letter() != 255 {
		t.Errorf(
			"New(255) created a T with letter %d, want %d", l2.Letter(), 255,
		)
	}

	// Test with special characters
	specialChars := []byte{
		'\n', '\t', '\r', ' ', '!', '"', '#', '$', '%', '&', '\'', '(', ')',
		'*', '+', ',', '-', '.', '/',
	}
	for _, c := range specialChars {
		l := New(c)
		if l.Letter() != c {
			t.Errorf(
				"New(%d) created a T with letter %d, want %d", c, l.Letter(), c,
			)
		}
	}
}
