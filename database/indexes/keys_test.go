package indexes

import (
	"bytes"
	"io"
	"orly.dev/chk"
	"orly.dev/codecbuf"
	"orly.dev/database/indexes/types"
	"testing"
)

// TestNewPrefix tests the NewPrefix function with and without arguments
func TestNewPrefix(t *testing.T) {
	// Test with no arguments (default prefix)
	defaultPrefix := NewPrefix()
	if len(defaultPrefix.Bytes()) != 3 {
		t.Errorf(
			"Default prefix should be 3 bytes, got %d",
			len(defaultPrefix.Bytes()),
		)
	}

	// Test with a valid prefix index
	validPrefix := NewPrefix(Event)
	if string(validPrefix.Bytes()) != string(EventPrefix) {
		t.Errorf("Expected prefix %q, got %q", EventPrefix, validPrefix.Bytes())
	}

	// Test with an invalid prefix index (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewPrefix should panic with invalid prefix index")
		}
	}()
	_ = NewPrefix(-1) // This should panic
}

// TestPrefixMethods tests the methods of the P struct
func TestPrefixMethods(t *testing.T) {
	// Create a prefix
	prefix := NewPrefix(Event)

	// Test Bytes method
	if !bytes.Equal(prefix.Bytes(), []byte(EventPrefix)) {
		t.Errorf(
			"Bytes method returned %v, expected %v", prefix.Bytes(),
			[]byte(EventPrefix),
		)
	}

	// Test MarshalWrite method
	buf := codecbuf.Get()
	err := prefix.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), []byte(EventPrefix)) {
		t.Errorf(
			"MarshalWrite wrote %v, expected %v", buf.Bytes(),
			[]byte(EventPrefix),
		)
	}

	// Test UnmarshalRead method
	newPrefix := &P{}
	err = newPrefix.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}
	if !bytes.Equal(newPrefix.Bytes(), []byte(EventPrefix)) {
		t.Errorf(
			"UnmarshalRead read %v, expected %v", newPrefix.Bytes(),
			[]byte(EventPrefix),
		)
	}
}

// TestPrefixFunction tests the Prefix function
func TestPrefixFunction(t *testing.T) {
	testCases := []struct {
		name     string
		index    int
		expected I
	}{
		{"Event", Event, EventPrefix},
		{"Id", Id, IdPrefix},
		{"IdPubkeyCreatedAt", IdPubkeyCreatedAt, IdPubkeyCreatedAtPrefix},
		{"Pubkey", Pubkey, PubkeyPrefix},
		{"PubkeyCreatedAt", PubkeyCreatedAt, PubkeyCreatedAtPrefix},
		{"CreatedAt", CreatedAt, CreatedAtPrefix},
		{"PubkeyTagCreatedAt", PubkeyTagCreatedAt, PubkeyTagCreatedAtPrefix},
		{"Tag", Tag, TagPrefix},
		{"TagCreatedAt", TagCreatedAt, TagCreatedAtPrefix},
		{"Kind", Kind, KindPrefix},
		{"KindCreatedAt", KindCreatedAt, KindCreatedAtPrefix},
		{"KindPubkey", KindPubkey, KindPubkeyPrefix},
		{"KindPubkeyCreatedAt", KindPubkeyCreatedAt, KindPubkeyCreatedAtPrefix},
		{"KindTag", KindTag, KindTagPrefix},
		{"KindTagCreatedAt", KindTagCreatedAt, KindTagCreatedAtPrefix},
		{
			"KindPubkeyTagCreatedAt", KindPubkeyTagCreatedAt,
			KindPubkeyTagCreatedAtPrefix,
		},
		{"Invalid", -1, ""},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				result := Prefix(tc.index)
				if result != tc.expected {
					t.Errorf(
						"Prefix(%d) = %q, expected %q", tc.index, result,
						tc.expected,
					)
				}
			},
		)
	}
}

// TestIdentify tests the Identify function
func TestIdentify(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   I
		expected int
	}{
		{"Event", EventPrefix, Event},
		{"Id", IdPrefix, Id},
		{"IdPubkeyCreatedAt", IdPubkeyCreatedAtPrefix, IdPubkeyCreatedAt},
		{"Pubkey", PubkeyPrefix, Pubkey},
		{"PubkeyCreatedAt", PubkeyCreatedAtPrefix, PubkeyCreatedAt},
		{"CreatedAt", CreatedAtPrefix, CreatedAt},
		{"PubkeyTagCreatedAt", PubkeyTagCreatedAtPrefix, PubkeyTagCreatedAt},
		{"Tag", TagPrefix, Tag},
		{"TagCreatedAt", TagCreatedAtPrefix, TagCreatedAt},
		{"Kind", KindPrefix, Kind},
		{"KindCreatedAt", KindCreatedAtPrefix, KindCreatedAt},
		{"KindPubkey", KindPubkeyPrefix, KindPubkey},
		{"KindPubkeyCreatedAt", KindPubkeyCreatedAtPrefix, KindPubkeyCreatedAt},
		{"KindTag", KindTagPrefix, KindTag},
		{"KindTagCreatedAt", KindTagCreatedAtPrefix, KindTagCreatedAt},
		{
			"KindPubkeyTagCreatedAt", KindPubkeyTagCreatedAtPrefix,
			KindPubkeyTagCreatedAt,
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				result, err := Identify(bytes.NewReader([]byte(tc.prefix)))
				if chk.E(err) {
					t.Fatalf("Identify failed: %v", err)
				}
				if result != tc.expected {
					t.Errorf(
						"Identify(%q) = %d, expected %d", tc.prefix, result,
						tc.expected,
					)
				}
			},
		)
	}

	// Test with invalid data
	t.Run(
		"Invalid", func(t *testing.T) {
			result, err := Identify(bytes.NewReader([]byte("xyz")))
			if chk.E(err) {
				t.Fatalf("Identify failed: %v", err)
			}
			if result != 0 {
				t.Errorf(
					"Identify with invalid prefix should return 0, got %d",
					result,
				)
			}
		},
	)

	// Test with error from reader
	t.Run(
		"ReaderError", func(t *testing.T) {
			errReader := &errorReader{}
			result, err := Identify(errReader)
			if err == nil {
				t.Errorf("Identify should return error with failing reader")
			}
			if result != -1 {
				t.Errorf(
					"Identify with reader error should return -1, got %d",
					result,
				)
			}
		},
	)
}

// errorReader is a mock reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// TestTStruct tests the T struct and its methods
func TestTStruct(t *testing.T) {
	// Create some test encoders
	prefix := NewPrefix(Event)
	ser := new(types.Uint40)
	ser.Set(12345)

	// Test New function
	enc := New(prefix, ser)
	if len(enc.Encs) != 2 {
		t.Errorf("New should create T with 2 encoders, got %d", len(enc.Encs))
	}

	// Test MarshalWrite
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Test UnmarshalRead
	dec := New(NewPrefix(), new(types.Uint40))
	err = dec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	decodedPrefix := dec.Encs[0].(*P)
	decodedSer := dec.Encs[1].(*types.Uint40)
	if !bytes.Equal(decodedPrefix.Bytes(), prefix.Bytes()) {
		t.Errorf(
			"Decoded prefix %v, expected %v", decodedPrefix.Bytes(),
			prefix.Bytes(),
		)
	}
	if decodedSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", decodedSer.Get(), ser.Get())
	}

	// Test with nil encoder
	encWithNil := New(prefix, nil, ser)
	buf.Reset()
	err = encWithNil.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite with nil encoder failed: %v", err)
	}
}

// TestEventFunctions tests the Event-related functions
func TestEventFunctions(t *testing.T) {
	// Test EventVars
	ser := EventVars()
	if ser == nil {
		t.Fatalf("EventVars should return non-nil *types.Uint40")
	}

	// Set a value
	ser.Set(12345)

	// Test EventEnc
	enc := EventEnc(ser)
	if len(enc.Encs) != 2 {
		t.Errorf(
			"EventEnc should create T with 2 encoders, got %d", len(enc.Encs),
		)
	}

	// Test EventDec
	dec := EventDec(ser)
	if len(dec.Encs) != 2 {
		t.Errorf(
			"EventDec should create T with 2 encoders, got %d", len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newSer := new(types.Uint40)
	newDec := EventDec(newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded value
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestIdFunctions tests the Id-related functions
func TestIdFunctions(t *testing.T) {
	// Test IdVars
	id, ser := IdVars()
	if id == nil || ser == nil {
		t.Fatalf("IdVars should return non-nil *types.IdHash and *types.Uint40")
	}

	// Set values
	id.Set([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	ser.Set(12345)

	// Test IdEnc
	enc := IdEnc(id, ser)
	if len(enc.Encs) != 3 {
		t.Errorf("IdEnc should create T with 3 encoders, got %d", len(enc.Encs))
	}

	// Test IdDec
	dec := IdDec(id, ser)
	if len(dec.Encs) != 3 {
		t.Errorf("IdDec should create T with 3 encoders, got %d", len(dec.Encs))
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newId, newSer := IdVars()
	newDec := IdDec(newId, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if !bytes.Equal(newId.Bytes(), id.Bytes()) {
		t.Errorf("Decoded id %v, expected %v", newId.Bytes(), id.Bytes())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestIdPubkeyCreatedAtFunctions tests the IdPubkeyCreatedAt-related functions
func TestIdPubkeyCreatedAtFunctions(t *testing.T) {
	// Test IdPubkeyCreatedAtVars
	ser, fid, p, ca := IdPubkeyCreatedAtVars()
	if ser == nil || fid == nil || p == nil || ca == nil {
		t.Fatalf("IdPubkeyCreatedAtVars should return non-nil values")
	}

	// Set values
	ser.Set(12345)
	err := fid.FromId(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromId failed: %v", err)
	}
	err = p.FromPubkey(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}
	ca.Set(98765)

	// Test IdPubkeyCreatedAtEnc
	enc := IdPubkeyCreatedAtEnc(ser, fid, p, ca)
	if len(enc.Encs) != 5 {
		t.Errorf(
			"IdPubkeyCreatedAtEnc should create T with 5 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test IdPubkeyCreatedAtDec
	dec := IdPubkeyCreatedAtDec(ser, fid, p, ca)
	if len(dec.Encs) != 5 {
		t.Errorf(
			"IdPubkeyCreatedAtDec should create T with 5 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newSer, newFid, newP, newCa := IdPubkeyCreatedAtVars()
	newDec := IdPubkeyCreatedAtDec(newSer, newFid, newP, newCa)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
	if !bytes.Equal(newFid.Bytes(), fid.Bytes()) {
		t.Errorf("Decoded id %v, expected %v", newFid.Bytes(), fid.Bytes())
	}
	if !bytes.Equal(newP.Bytes(), p.Bytes()) {
		t.Errorf("Decoded pubkey hash %v, expected %v", newP.Bytes(), p.Bytes())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
}

// TestCreatedAtFunctions tests the CreatedAt-related functions
func TestCreatedAtFunctions(t *testing.T) {
	// Test CreatedAtVars
	ca, ser := CreatedAtVars()
	if ca == nil || ser == nil {
		t.Fatalf("CreatedAtVars should return non-nil values")
	}

	// Set values
	ca.Set(98765)
	ser.Set(12345)

	// Test CreatedAtEnc
	enc := CreatedAtEnc(ca, ser)
	if len(enc.Encs) != 3 {
		t.Errorf(
			"CreatedAtEnc should create T with 3 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test CreatedAtDec
	dec := CreatedAtDec(ca, ser)
	if len(dec.Encs) != 3 {
		t.Errorf(
			"CreatedAtDec should create T with 3 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newCa, newSer := CreatedAtVars()
	newDec := CreatedAtDec(newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestPubkeyFunctions tests the Pubkey-related functions
func TestPubkeyFunctions(t *testing.T) {
	// Test PubkeyVars
	p, ser := PubkeyVars()
	if p == nil || ser == nil {
		t.Fatalf("PubkeyVars should return non-nil values")
	}

	// Set values
	err := p.FromPubkey(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}
	ser.Set(12345)

	// Test PubkeyEnc
	enc := PubkeyEnc(p, ser)
	if len(enc.Encs) != 3 {
		t.Errorf(
			"PubkeyEnc should create T with 3 encoders, got %d", len(enc.Encs),
		)
	}

	// Test PubkeyDec
	dec := PubkeyDec(p, ser)
	if len(dec.Encs) != 3 {
		t.Errorf(
			"PubkeyDec should create T with 3 encoders, got %d", len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newP, newSer := PubkeyVars()
	newDec := PubkeyDec(newP, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if !bytes.Equal(newP.Bytes(), p.Bytes()) {
		t.Errorf("Decoded pubkey hash %v, expected %v", newP.Bytes(), p.Bytes())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestPubkeyCreatedAtFunctions tests the PubkeyCreatedAt-related functions
func TestPubkeyCreatedAtFunctions(t *testing.T) {
	// Test PubkeyCreatedAtVars
	p, ca, ser := PubkeyCreatedAtVars()
	if p == nil || ca == nil || ser == nil {
		t.Fatalf("PubkeyCreatedAtVars should return non-nil values")
	}

	// Set values
	err := p.FromPubkey(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}
	ca.Set(98765)
	ser.Set(12345)

	// Test PubkeyCreatedAtEnc
	enc := PubkeyCreatedAtEnc(p, ca, ser)
	if len(enc.Encs) != 4 {
		t.Errorf(
			"PubkeyCreatedAtEnc should create T with 4 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test PubkeyCreatedAtDec
	dec := PubkeyCreatedAtDec(p, ca, ser)
	if len(dec.Encs) != 4 {
		t.Errorf(
			"PubkeyCreatedAtDec should create T with 4 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newP, newCa, newSer := PubkeyCreatedAtVars()
	newDec := PubkeyCreatedAtDec(newP, newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if !bytes.Equal(newP.Bytes(), p.Bytes()) {
		t.Errorf("Decoded pubkey hash %v, expected %v", newP.Bytes(), p.Bytes())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestPubkeyTagCreatedAtFunctions tests the PubkeyTagCreatedAt-related functions
func TestPubkeyTagCreatedAtFunctions(t *testing.T) {
	// Test PubkeyTagCreatedAtVars
	p, k, v, ca, ser := PubkeyTagCreatedAtVars()
	if p == nil || k == nil || v == nil || ca == nil || ser == nil {
		t.Fatalf("PubkeyTagCreatedAtVars should return non-nil values")
	}

	// Set values
	err := p.FromPubkey(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}
	k.Set('e')
	err = v.FromIdent([]byte("test-value"))
	if chk.E(err) {
		t.Fatalf("FromIdent failed: %v", err)
	}
	ca.Set(98765)
	ser.Set(12345)

	// Test PubkeyTagCreatedAtEnc
	enc := PubkeyTagCreatedAtEnc(p, k, v, ca, ser)
	if len(enc.Encs) != 6 {
		t.Errorf(
			"PubkeyTagCreatedAtEnc should create T with 6 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test PubkeyTagCreatedAtDec
	dec := PubkeyTagCreatedAtDec(p, k, v, ca, ser)
	if len(dec.Encs) != 6 {
		t.Errorf(
			"PubkeyTagCreatedAtDec should create T with 6 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newP, newK, newV, newCa, newSer := PubkeyTagCreatedAtVars()
	newDec := PubkeyTagCreatedAtDec(newP, newK, newV, newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if !bytes.Equal(newP.Bytes(), p.Bytes()) {
		t.Errorf("Decoded pubkey hash %v, expected %v", newP.Bytes(), p.Bytes())
	}
	if newK.Letter() != k.Letter() {
		t.Errorf(
			"Decoded key letter %c, expected %c", newK.Letter(), k.Letter(),
		)
	}
	if !bytes.Equal(newV.Bytes(), v.Bytes()) {
		t.Errorf("Decoded value hash %v, expected %v", newV.Bytes(), v.Bytes())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestTagFunctions tests the Tag-related functions
func TestTagFunctions(t *testing.T) {
	// Test TagVars
	k, v, ser := TagVars()
	if k == nil || v == nil || ser == nil {
		t.Fatalf("TagVars should return non-nil values")
	}

	// Set values
	k.Set('e')
	err := v.FromIdent([]byte("test-value"))
	if chk.E(err) {
		t.Fatalf("FromIdent failed: %v", err)
	}
	ser.Set(12345)

	// Test TagEnc
	enc := TagEnc(k, v, ser)
	if len(enc.Encs) != 4 {
		t.Errorf(
			"TagEnc should create T with 4 encoders, got %d", len(enc.Encs),
		)
	}

	// Test TagDec
	dec := TagDec(k, v, ser)
	if len(dec.Encs) != 4 {
		t.Errorf(
			"TagDec should create T with 4 encoders, got %d", len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newK, newV, newSer := TagVars()
	newDec := TagDec(newK, newV, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newK.Letter() != k.Letter() {
		t.Errorf(
			"Decoded key letter %c, expected %c", newK.Letter(), k.Letter(),
		)
	}
	if !bytes.Equal(newV.Bytes(), v.Bytes()) {
		t.Errorf("Decoded value hash %v, expected %v", newV.Bytes(), v.Bytes())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestTagCreatedAtFunctions tests the TagCreatedAt-related functions
func TestTagCreatedAtFunctions(t *testing.T) {
	// Test TagCreatedAtVars
	k, v, ca, ser := TagCreatedAtVars()
	if k == nil || v == nil || ca == nil || ser == nil {
		t.Fatalf("TagCreatedAtVars should return non-nil values")
	}

	// Set values
	k.Set('e')
	err := v.FromIdent([]byte("test-value"))
	if chk.E(err) {
		t.Fatalf("FromIdent failed: %v", err)
	}
	ca.Set(98765)
	ser.Set(12345)

	// Test TagCreatedAtEnc
	enc := TagCreatedAtEnc(k, v, ca, ser)
	if len(enc.Encs) != 5 {
		t.Errorf(
			"TagCreatedAtEnc should create T with 5 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test TagCreatedAtDec
	dec := TagCreatedAtDec(k, v, ca, ser)
	if len(dec.Encs) != 5 {
		t.Errorf(
			"TagCreatedAtDec should create T with 5 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newK, newV, newCa, newSer := TagCreatedAtVars()
	newDec := TagCreatedAtDec(newK, newV, newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newK.Letter() != k.Letter() {
		t.Errorf(
			"Decoded key letter %c, expected %c", newK.Letter(), k.Letter(),
		)
	}
	if !bytes.Equal(newV.Bytes(), v.Bytes()) {
		t.Errorf("Decoded value hash %v, expected %v", newV.Bytes(), v.Bytes())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestKindFunctions tests the Kind-related functions
func TestKindFunctions(t *testing.T) {
	// Test KindVars
	ki, ser := KindVars()
	if ki == nil || ser == nil {
		t.Fatalf("KindVars should return non-nil values")
	}

	// Set values
	ki.Set(1234)
	ser.Set(12345)

	// Test KindEnc
	enc := KindEnc(ki, ser)
	if len(enc.Encs) != 3 {
		t.Errorf(
			"KindEnc should create T with 3 encoders, got %d", len(enc.Encs),
		)
	}

	// Test KindDec
	dec := KindDec(ki, ser)
	if len(dec.Encs) != 3 {
		t.Errorf(
			"KindDec should create T with 3 encoders, got %d", len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newKi, newSer := KindVars()
	newDec := KindDec(newKi, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newKi.Get() != ki.Get() {
		t.Errorf("Decoded kind %d, expected %d", newKi.Get(), ki.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestKindPubkeyFunctions tests the KindPubkey-related functions
func TestKindPubkeyFunctions(t *testing.T) {
	// Test KindPubkeyVars
	ki, p, ser := KindPubkeyVars()
	if ki == nil || p == nil || ser == nil {
		t.Fatalf("KindPubkeyVars should return non-nil values")
	}

	// Set values
	ki.Set(1234)
	err := p.FromPubkey(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}
	ser.Set(12345)

	// Test KindPubkeyEnc
	enc := KindPubkeyEnc(ki, p, ser)
	if len(enc.Encs) != 4 {
		t.Errorf(
			"KindPubkeyEnc should create T with 4 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test KindPubkeyDec
	dec := KindPubkeyDec(ki, p, ser)
	if len(dec.Encs) != 4 {
		t.Errorf(
			"KindPubkeyDec should create T with 4 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newKi, newP, newSer := KindPubkeyVars()
	newDec := KindPubkeyDec(newKi, newP, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newKi.Get() != ki.Get() {
		t.Errorf("Decoded kind %d, expected %d", newKi.Get(), ki.Get())
	}
	if !bytes.Equal(newP.Bytes(), p.Bytes()) {
		t.Errorf("Decoded pubkey hash %v, expected %v", newP.Bytes(), p.Bytes())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestKindCreatedAtFunctions tests the KindCreatedAt-related functions
func TestKindCreatedAtFunctions(t *testing.T) {
	// Test KindCreatedAtVars
	ki, ca, ser := KindCreatedAtVars()
	if ki == nil || ca == nil || ser == nil {
		t.Fatalf("KindCreatedAtVars should return non-nil values")
	}

	// Set values
	ki.Set(1234)
	ca.Set(98765)
	ser.Set(12345)

	// Test KindCreatedAtEnc
	enc := KindCreatedAtEnc(ki, ca, ser)
	if len(enc.Encs) != 4 {
		t.Errorf(
			"KindCreatedAtEnc should create T with 4 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test KindCreatedAtDec
	dec := KindCreatedAtDec(ki, ca, ser)
	if len(dec.Encs) != 4 {
		t.Errorf(
			"KindCreatedAtDec should create T with 4 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err := enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newKi, newCa, newSer := KindCreatedAtVars()
	newDec := KindCreatedAtDec(newKi, newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newKi.Get() != ki.Get() {
		t.Errorf("Decoded kind %d, expected %d", newKi.Get(), ki.Get())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestKindTagFunctions tests the KindTag-related functions
func TestKindTagFunctions(t *testing.T) {
	// Test KindTagVars
	ki, k, v, ser := KindTagVars()
	if ki == nil || k == nil || v == nil || ser == nil {
		t.Fatalf("KindTagVars should return non-nil values")
	}

	// Set values
	ki.Set(1234)
	k.Set('e')
	err := v.FromIdent([]byte("test-value"))
	if chk.E(err) {
		t.Fatalf("FromIdent failed: %v", err)
	}
	ser.Set(12345)

	// Test KindTagEnc
	enc := KindTagEnc(ki, k, v, ser)
	if len(enc.Encs) != 5 {
		t.Errorf(
			"KindTagEnc should create T with 5 encoders, got %d", len(enc.Encs),
		)
	}

	// Test KindTagDec
	dec := KindTagDec(ki, k, v, ser)
	if len(dec.Encs) != 5 {
		t.Errorf(
			"KindTagDec should create T with 5 encoders, got %d", len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newKi, newK, newV, newSer := KindTagVars()
	newDec := KindTagDec(newKi, newK, newV, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newKi.Get() != ki.Get() {
		t.Errorf("Decoded kind %d, expected %d", newKi.Get(), ki.Get())
	}
	if newK.Letter() != k.Letter() {
		t.Errorf(
			"Decoded key letter %c, expected %c", newK.Letter(), k.Letter(),
		)
	}
	if !bytes.Equal(newV.Bytes(), v.Bytes()) {
		t.Errorf("Decoded value hash %v, expected %v", newV.Bytes(), v.Bytes())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestKindTagCreatedAtFunctions tests the KindTagCreatedAt-related functions
func TestKindTagCreatedAtFunctions(t *testing.T) {
	// Test KindTagCreatedAtVars
	ki, k, v, ca, ser := KindTagCreatedAtVars()
	if ki == nil || k == nil || v == nil || ca == nil || ser == nil {
		t.Fatalf("KindTagCreatedAtVars should return non-nil values")
	}

	// Set values
	ki.Set(1234)
	k.Set('e')
	err := v.FromIdent([]byte("test-value"))
	if chk.E(err) {
		t.Fatalf("FromIdent failed: %v", err)
	}
	ca.Set(98765)
	ser.Set(12345)

	// Test KindTagCreatedAtEnc
	enc := KindTagCreatedAtEnc(ki, k, v, ca, ser)
	if len(enc.Encs) != 6 {
		t.Errorf(
			"KindTagCreatedAtEnc should create T with 6 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test KindTagCreatedAtDec
	dec := KindTagCreatedAtDec(ki, k, v, ca, ser)
	if len(dec.Encs) != 6 {
		t.Errorf(
			"KindTagCreatedAtDec should create T with 6 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newKi, newK, newV, newCa, newSer := KindTagCreatedAtVars()
	newDec := KindTagCreatedAtDec(newKi, newK, newV, newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newKi.Get() != ki.Get() {
		t.Errorf("Decoded kind %d, expected %d", newKi.Get(), ki.Get())
	}
	if newK.Letter() != k.Letter() {
		t.Errorf(
			"Decoded key letter %c, expected %c", newK.Letter(), k.Letter(),
		)
	}
	if !bytes.Equal(newV.Bytes(), v.Bytes()) {
		t.Errorf("Decoded value hash %v, expected %v", newV.Bytes(), v.Bytes())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestKindPubkeyCreatedAtFunctions tests the KindPubkeyCreatedAt-related functions
func TestKindPubkeyCreatedAtFunctions(t *testing.T) {
	// Test KindPubkeyCreatedAtVars
	ki, p, ca, ser := KindPubkeyCreatedAtVars()
	if ki == nil || p == nil || ca == nil || ser == nil {
		t.Fatalf("KindPubkeyCreatedAtVars should return non-nil values")
	}

	// Set values
	ki.Set(1234)
	err := p.FromPubkey(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}
	ca.Set(98765)
	ser.Set(12345)

	// Test KindPubkeyCreatedAtEnc
	enc := KindPubkeyCreatedAtEnc(ki, p, ca, ser)
	if len(enc.Encs) != 5 {
		t.Errorf(
			"KindPubkeyCreatedAtEnc should create T with 5 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test KindPubkeyCreatedAtDec
	dec := KindPubkeyCreatedAtDec(ki, p, ca, ser)
	if len(dec.Encs) != 5 {
		t.Errorf(
			"KindPubkeyCreatedAtDec should create T with 5 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newKi, newP, newCa, newSer := KindPubkeyCreatedAtVars()
	newDec := KindPubkeyCreatedAtDec(newKi, newP, newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newKi.Get() != ki.Get() {
		t.Errorf("Decoded kind %d, expected %d", newKi.Get(), ki.Get())
	}
	if !bytes.Equal(newP.Bytes(), p.Bytes()) {
		t.Errorf("Decoded pubkey hash %v, expected %v", newP.Bytes(), p.Bytes())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}

// TestKindPubkeyTagCreatedAtFunctions tests the KindPubkeyTagCreatedAt-related functions
func TestKindPubkeyTagCreatedAtFunctions(t *testing.T) {
	// Test KindPubkeyTagCreatedAtVars
	ki, p, k, v, ca, ser := KindPubkeyTagCreatedAtVars()
	if ki == nil || p == nil || k == nil || v == nil || ca == nil || ser == nil {
		t.Fatalf("KindPubkeyTagCreatedAtVars should return non-nil values")
	}

	// Set values
	ki.Set(1234)
	err := p.FromPubkey(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		},
	)
	if chk.E(err) {
		t.Fatalf("FromPubkey failed: %v", err)
	}
	k.Set('e')
	err = v.FromIdent([]byte("test-value"))
	if chk.E(err) {
		t.Fatalf("FromIdent failed: %v", err)
	}
	ca.Set(98765)
	ser.Set(12345)

	// Test KindPubkeyTagCreatedAtEnc
	enc := KindPubkeyTagCreatedAtEnc(ki, p, k, v, ca, ser)
	if len(enc.Encs) != 7 {
		t.Errorf(
			"KindPubkeyTagCreatedAtEnc should create T with 7 encoders, got %d",
			len(enc.Encs),
		)
	}

	// Test KindPubkeyTagCreatedAtDec
	dec := KindPubkeyTagCreatedAtDec(ki, p, k, v, ca, ser)
	if len(dec.Encs) != 7 {
		t.Errorf(
			"KindPubkeyTagCreatedAtDec should create T with 7 encoders, got %d",
			len(dec.Encs),
		)
	}

	// Test marshaling and unmarshaling
	buf := codecbuf.Get()
	err = enc.MarshalWrite(buf)
	if chk.E(err) {
		t.Fatalf("MarshalWrite failed: %v", err)
	}

	// Create new variables for decoding
	newKi, newP, newK, newV, newCa, newSer := KindPubkeyTagCreatedAtVars()
	newDec := KindPubkeyTagCreatedAtDec(newKi, newP, newK, newV, newCa, newSer)

	err = newDec.UnmarshalRead(bytes.NewBuffer(buf.Bytes()))
	if chk.E(err) {
		t.Fatalf("UnmarshalRead failed: %v", err)
	}

	// Verify the decoded values
	if newKi.Get() != ki.Get() {
		t.Errorf("Decoded kind %d, expected %d", newKi.Get(), ki.Get())
	}
	if !bytes.Equal(newP.Bytes(), p.Bytes()) {
		t.Errorf("Decoded pubkey hash %v, expected %v", newP.Bytes(), p.Bytes())
	}
	if newK.Letter() != k.Letter() {
		t.Errorf(
			"Decoded key letter %c, expected %c", newK.Letter(), k.Letter(),
		)
	}
	if !bytes.Equal(newV.Bytes(), v.Bytes()) {
		t.Errorf("Decoded value hash %v, expected %v", newV.Bytes(), v.Bytes())
	}
	if newCa.Get() != ca.Get() {
		t.Errorf("Decoded created at %d, expected %d", newCa.Get(), ca.Get())
	}
	if newSer.Get() != ser.Get() {
		t.Errorf("Decoded serial %d, expected %d", newSer.Get(), ser.Get())
	}
}
