package indexes

import (
	"bytes"
	"not.realy.lol/codecbuf"
	"testing"

	"not.realy.lol/chk"
	"not.realy.lol/database/indexes/types/idhash"
	. "not.realy.lol/database/indexes/types/number"
)

func TestNext(t *testing.T) {
	// Save the current counter value
	initialCounter := counter

	// Call next() and verify it increments the counter
	val1 := next()
	if val1 != initialCounter {
		t.Errorf("Expected next() to return %d, got %d", initialCounter, val1)
	}

	// Call next() again and verify it increments the counter again
	val2 := next()
	if val2 != initialCounter+1 {
		t.Errorf("Expected next() to return %d, got %d", initialCounter+1, val2)
	}

	// Reset the counter to its initial value to avoid affecting other tests
	counter = initialCounter
}

func TestPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prf      int
		expected I
	}{
		{"Event", Event, "evt"},
		{"Id", Id, "eid"},
		{"IdPubkeyCreatedAt", IdPubkeyCreatedAt, "ipc"},
		{"PubkeyCreatedAt", PubkeyCreatedAt, "pca"},
		{"CreatedAt", CreatedAt, "ica"},
		{"PubkeyTagCreatedAt", PubkeyTagCreatedAt, "ptc"},
		{"TagCreatedAt", TagCreatedAt, "itc"},
		{"Kind", Kind, "iki"},
		{"KindCreatedAt", KindCreatedAt, "kca"},
		{"KindPubkey", KindPubkey, "kpk"},
		{"KindPubkeyCreatedAt", KindPubkeyCreatedAt, "kpc"},
		{"KindTag", KindTag, "ikt"},
		{"KindTagCreatedAt", KindTagCreatedAt, "ktc"},
		{"KindPubkeyTagCreatedAt", KindPubkeyTagCreatedAt, "kpt"},
		{"Unknown", 999, ""}, // Test an unknown prefix
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := Prefix(tt.prf)
				if result != tt.expected {
					t.Errorf(
						"Prefix(%d) = %s, want %s", tt.prf, result, tt.expected,
					)
				}
			},
		)
	}
}

func TestNewPrefix(t *testing.T) {
	// Test with no arguments
	p1 := NewPrefix()
	if len(p1.val) != 3 || p1.val[0] != 0 || p1.val[1] != 0 || p1.val[2] != 0 {
		t.Errorf("NewPrefix() = %v, want [0 0 0]", p1.val)
	}

	// Test with a prefix argument
	p2 := NewPrefix(Event)
	expected := []byte(Prefix(Event))
	if !bytes.Equal(p2.val, expected) {
		t.Errorf("NewPrefix(%d) = %v, want %v", Event, p2.val, expected)
	}
}

func TestPMarshalWriteUnmarshalRead(t *testing.T) {
	// Create a prefix
	p := NewPrefix(Event)

	// Test MarshalWrite
	buf := codecbuf.Get()
	err := p.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create a new prefix for unmarshaling
	p2 := NewPrefix(Event) // Use the same prefix to ensure the buffer has the right size

	// Test UnmarshalRead
	buf2 := bytes.NewBuffer(buf.Bytes())
	err = p2.UnmarshalRead(buf2)
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the unmarshaled value matches the original
	if !bytes.Equal(p.val, p2.val) {
		t.Errorf(
			"Unmarshaled value %v does not match original %v", p2.val, p.val,
		)
	}
}

func TestIWrite(t *testing.T) {
	// Create an I
	i := I("test")

	// Test Write
	buf := codecbuf.Get()
	n, err := i.Write(buf)
	if chk.E(err) {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify the number of bytes written
	if n != 4 {
		t.Errorf("Write returned %d, want 4", n)
	}

	// Verify the written bytes
	if buf.String() != "test" {
		t.Errorf("Write wrote %s, want test", buf.String())
	}
}

func TestTMarshalWriteUnmarshalRead(t *testing.T) {
	// Create encoders
	ser := new(Uint40)
	ser.Set(12345)

	// Create a T
	enc := EventEnc(ser)

	// Test MarshalWrite
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create a new T for unmarshaling
	ser2 := new(Uint40)
	enc2 := EventDec(ser2)

	// Test UnmarshalRead
	buf2 := bytes.NewBuffer(buf.Bytes())
	err = enc2.UnmarshalRead(buf2)
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the unmarshaled value matches the original
	if ser2.Get() != ser.Get() {
		t.Errorf(
			"Unmarshaled value %d does not match original %d", ser2.Get(),
			ser.Get(),
		)
	}
}

func TestEventFunctions(t *testing.T) {
	// Test EventVars
	ser := EventVars()
	if ser == nil {
		t.Fatal("EventVars() returned nil")
	}

	// Set a value
	ser.Set(12345)

	// Test EventEnc
	enc := EventEnc(ser)
	if enc == nil {
		t.Fatal("EventEnc() returned nil")
	}

	// Test MarshalWrite
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Test EventDec
	ser2 := new(Uint40)
	dec := EventDec(ser2)
	if dec == nil {
		t.Fatal("EventDec() returned nil")
	}

	// Test UnmarshalRead
	buf2 := bytes.NewBuffer(buf.Bytes())
	err = dec.UnmarshalRead(buf2)
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the unmarshaled value matches the original
	if ser2.Get() != ser.Get() {
		t.Errorf(
			"Unmarshaled value %d does not match original %d", ser2.Get(),
			ser.Get(),
		)
	}
}

func TestIdFunctions(t *testing.T) {
	// Test IdVars
	id, ser := IdVars()
	if id == nil || ser == nil {
		t.Fatal("IdVars() returned nil")
	}

	// Set values
	err := id.FromIdHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if chk.E(err) {
		t.Fatalf("FromIdHex failed: %v", err)
	}
	ser.Set(12345)

	// Test IdEnc
	enc := IdEnc(id, ser)
	if enc == nil {
		t.Fatal("IdEnc() returned nil")
	}

	// Test MarshalWrite
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Test IdSearch
	search := IdSearch(id)
	if search == nil {
		t.Fatal("IdSearch() returned nil")
	}

	// Test IdDec
	id2 := idhash.New()
	ser2 := new(Uint40)
	dec := IdDec(id2, ser2)
	if dec == nil {
		t.Fatal("IdDec() returned nil")
	}

	// Test UnmarshalRead
	buf2 := bytes.NewBuffer(buf.Bytes())
	err = dec.UnmarshalRead(buf2)
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the unmarshaled values match the originals
	if !bytes.Equal(id2.Bytes(), id.Bytes()) {
		t.Errorf(
			"Unmarshaled id %v does not match original %v", id2.Bytes(),
			id.Bytes(),
		)
	}
	if ser2.Get() != ser.Get() {
		t.Errorf(
			"Unmarshaled ser %d does not match original %d", ser2.Get(),
			ser.Get(),
		)
	}
}
