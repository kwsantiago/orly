package types

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestTypesSortLexicographically tests if the numeric types sort lexicographically
// when using bytes.Compare after marshaling.
func TestTypesSortLexicographically(t *testing.T) {
	// Test Uint16
	t.Run("Uint16", func(t *testing.T) {
		testUint16Sorting(t)
	})

	// Test Uint24
	t.Run("Uint24", func(t *testing.T) {
		testUint24Sorting(t)
	})

	// Test Uint32
	t.Run("Uint32", func(t *testing.T) {
		testUint32Sorting(t)
	})

	// Test Uint40
	t.Run("Uint40", func(t *testing.T) {
		testUint40Sorting(t)
	})

	// Test Uint64
	t.Run("Uint64", func(t *testing.T) {
		testUint64Sorting(t)
	})
}

// TestEdgeCases tests sorting with edge cases like zero, max values, and adjacent values
func TestEdgeCases(t *testing.T) {
	// Test Uint16 edge cases
	t.Run("Uint16EdgeCases", func(t *testing.T) {
		testUint16EdgeCases(t)
	})

	// Test Uint24 edge cases
	t.Run("Uint24EdgeCases", func(t *testing.T) {
		testUint24EdgeCases(t)
	})

	// Test Uint32 edge cases
	t.Run("Uint32EdgeCases", func(t *testing.T) {
		testUint32EdgeCases(t)
	})

	// Test Uint40 edge cases
	t.Run("Uint40EdgeCases", func(t *testing.T) {
		testUint40EdgeCases(t)
	})

	// Test Uint64 edge cases
	t.Run("Uint64EdgeCases", func(t *testing.T) {
		testUint64EdgeCases(t)
	})
}

func testUint16Sorting(t *testing.T) {
	values := []uint16{1, 10, 100, 1000, 10000, 65535}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint16)
		u.Set(val)

		buf := new(bytes.Buffer)
		err := u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint16 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint16 values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint24Sorting(t *testing.T) {
	values := []uint32{1, 10, 100, 1000, 10000, 100000, 1000000, 16777215}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint24)
		err := u.Set(val)
		if err != nil {
			t.Fatalf("Failed to set Uint24 %d: %v", val, err)
		}

		buf := new(bytes.Buffer)
		err = u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint24 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint24 values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint32Sorting(t *testing.T) {
	values := []uint32{1, 10, 100, 1000, 10000, 100000, 1000000, 4294967295}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint32)
		u.Set(val)

		buf := new(bytes.Buffer)
		err := u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint32 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint32 values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint40Sorting(t *testing.T) {
	values := []uint64{1, 10, 100, 1000, 10000, 100000, 1000000, 1099511627775}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint40)
		err := u.Set(val)
		if err != nil {
			t.Fatalf("Failed to set Uint40 %d: %v", val, err)
		}

		buf := new(bytes.Buffer)
		err = u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint40 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint40 values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint64Sorting(t *testing.T) {
	values := []uint64{1, 10, 100, 1000, 10000, 100000, 1000000, 18446744073709551615}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint64)
		u.Set(val)

		buf := new(bytes.Buffer)
		err := u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint64 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint64 values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

// Edge case test functions

func testUint16EdgeCases(t *testing.T) {
	// Test edge cases: 0, max value, and adjacent values
	values := []uint16{0, 1, 2, 65534, 65535}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint16)
		u.Set(val)

		buf := new(bytes.Buffer)
		err := u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint16 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint16 edge case values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint24EdgeCases(t *testing.T) {
	// Test edge cases: 0, max value, and adjacent values
	values := []uint32{0, 1, 2, 16777214, 16777215}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint24)
		err := u.Set(val)
		if err != nil {
			t.Fatalf("Failed to set Uint24 %d: %v", val, err)
		}

		buf := new(bytes.Buffer)
		err = u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint24 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint24 edge case values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint32EdgeCases(t *testing.T) {
	// Test edge cases: 0, max value, and adjacent values
	values := []uint32{0, 1, 2, 4294967294, 4294967295}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint32)
		u.Set(val)

		buf := new(bytes.Buffer)
		err := u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint32 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint32 edge case values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint40EdgeCases(t *testing.T) {
	// Test edge cases: 0, max value, and adjacent values
	values := []uint64{0, 1, 2, 1099511627774, 1099511627775}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint40)
		err := u.Set(val)
		if err != nil {
			t.Fatalf("Failed to set Uint40 %d: %v", val, err)
		}

		buf := new(bytes.Buffer)
		err = u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint40 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint40 edge case values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

func testUint64EdgeCases(t *testing.T) {
	// Test edge cases: 0, max value, and adjacent values
	values := []uint64{0, 1, 2, 18446744073709551614, 18446744073709551615}

	// Marshal each value
	marshaledValues := make([][]byte, len(values))
	for i, val := range values {
		u := new(Uint64)
		u.Set(val)

		buf := new(bytes.Buffer)
		err := u.MarshalWrite(buf)
		if err != nil {
			t.Fatalf("Failed to marshal Uint64 %d: %v", val, err)
		}

		marshaledValues[i] = buf.Bytes()
	}

	// Check if they sort correctly with bytes.Compare
	for i := 0; i < len(marshaledValues)-1; i++ {
		if bytes.Compare(marshaledValues[i], marshaledValues[i+1]) >= 0 {
			t.Errorf("Uint64 edge case values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", marshaledValues[i], marshaledValues[i+1])
		}
	}
}

// TestEndianness demonstrates why BigEndian is used instead of LittleEndian
// for lexicographical sorting with bytes.Compare
func TestEndianness(t *testing.T) {
	// Test with uint32 values
	values := []uint32{1, 10, 100, 1000, 10000}

	// Marshal each value using BigEndian
	bigEndianValues := make([][]byte, len(values))
	for i, val := range values {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, val)
		bigEndianValues[i] = buf
	}

	// Marshal each value using LittleEndian
	littleEndianValues := make([][]byte, len(values))
	for i, val := range values {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, val)
		littleEndianValues[i] = buf
	}

	// Check if BigEndian values sort correctly with bytes.Compare
	t.Log("Testing BigEndian sorting:")
	for i := 0; i < len(bigEndianValues)-1; i++ {
		result := bytes.Compare(bigEndianValues[i], bigEndianValues[i+1])
		t.Logf("Compare %d with %d: result = %d", values[i], values[i+1], result)
		if result >= 0 {
			t.Errorf("BigEndian values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", bigEndianValues[i], bigEndianValues[i+1])
		}
	}

	// Check if LittleEndian values sort correctly with bytes.Compare
	t.Log("Testing LittleEndian sorting:")
	correctOrder := true
	for i := 0; i < len(littleEndianValues)-1; i++ {
		result := bytes.Compare(littleEndianValues[i], littleEndianValues[i+1])
		t.Logf("Compare %d with %d: result = %d", values[i], values[i+1], result)
		if result >= 0 {
			correctOrder = false
			t.Logf("LittleEndian values don't sort correctly: %v should be less than %v",
				values[i], values[i+1])
			t.Logf("Bytes representation: %v vs %v", littleEndianValues[i], littleEndianValues[i+1])
		}
	}

	// We expect LittleEndian to NOT sort correctly
	if correctOrder {
		t.Error("LittleEndian values unexpectedly sorted correctly")
	} else {
		t.Log("As expected, LittleEndian values don't sort correctly with bytes.Compare")
	}
}
