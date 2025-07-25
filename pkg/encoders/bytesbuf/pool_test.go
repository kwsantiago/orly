package bytesbuf

import (
	"bytes"
	"testing"
)

func TestPool(t *testing.T) {
	// Create a new pool
	pool := NewPool()

	// Get a buffer from the pool
	buf := pool.Get()
	if buf == nil {
		t.Fatal("Expected non-nil buffer from pool")
	}

	// Write some data to the buffer
	testData := []byte("test data")
	buf = append(buf, testData...)

	// Verify the buffer contains the expected data
	if !bytes.Equal(buf, testData) {
		t.Fatalf(
			"Expected buffer to contain %q, got %q", testData, buf,
		)
	}

	// Put the buffer back in the pool
	pool.Put(buf)

	// Get another buffer from the pool (should be the same one, reset)
	buf2 := pool.Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from pool")
	}

	// Verify the buffer is empty (was reset)
	if len(buf2) != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", len(buf2))
	}

	// Write different data to the buffer
	testData2 := []byte("different data")
	buf2 = append(buf2, testData2...)

	// Verify the buffer contains the new data
	if !bytes.Equal(buf2, testData2) {
		t.Fatalf(
			"Expected buffer to contain %q, got %q", testData2, buf2,
		)
	}
}

func TestDefaultPool(t *testing.T) {
	// Get a buffer from the default pool
	buf := Get()
	if buf == nil {
		t.Fatal("Expected non-nil buffer from default pool")
	}

	// Write some data to the buffer
	testData := []byte("test data for default pool")
	buf = append(buf, testData...)

	// Verify the buffer contains the expected data
	if !bytes.Equal(buf, testData) {
		t.Fatalf(
			"Expected buffer to contain %q, got %q", testData, buf,
		)
	}

	// Put the buffer back in the pool
	Put(buf)

	// Get another buffer from the pool (should be reset)
	buf2 := Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from default pool")
	}

	// Verify the buffer is empty (was reset)
	if len(buf2) != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", len(buf2))
	}
}

func TestZeroBytes(t *testing.T) {
	// Create a new pool
	pool := NewPool()

	// Get a buffer from the pool
	buf := pool.Get()
	if buf == nil {
		t.Fatal("Expected non-nil buffer from pool")
	}

	// Write some sensitive data to the buffer
	sensitiveData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	buf = append(buf, sensitiveData...)

	// Get the capacity before putting it back
	capacity := cap(buf)

	// Put the buffer back in the pool
	pool.Put(buf)

	// Get another buffer from the pool (should be the same one, reset)
	buf2 := pool.Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from pool")
	}

	// Verify the buffer is empty (was reset)
	if len(buf2) != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", len(buf2))
	}

	// Verify the capacity is at least the same (should be the same buffer)
	if cap(buf2) < capacity {
		t.Fatalf("Expected capacity at least %d, got %d", capacity, cap(buf2))
	}

	// Grow the buffer to expose the underlying memory
	buf2 = append(buf2, make([]byte, len(sensitiveData))...)

	// Write some new data to the buffer to expose the underlying memory
	newData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	copy(buf2, newData)

	// Verify that the sensitive data was zeroed out
	// The new data should be there, but no trace of the old data
	for i, b := range buf2[:len(newData)] {
		if b != newData[i] {
			t.Fatalf("Expected byte %d to be %d, got %d", i, newData[i], b)
		}
	}
}

func TestDefaultPoolZeroBytes(t *testing.T) {
	// Get a buffer from the default pool
	buf := Get()
	if buf == nil {
		t.Fatal("Expected non-nil buffer from default pool")
	}

	// Write some sensitive data to the buffer
	sensitiveData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	buf = append(buf, sensitiveData...)

	// Get the capacity before putting it back
	capacity := cap(buf)

	// Put the buffer back in the pool
	Put(buf)

	// Get another buffer from the pool (should be the same one, reset)
	buf2 := Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from default pool")
	}

	// Verify the buffer is empty (was reset)
	if len(buf2) != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", len(buf2))
	}

	// Verify the capacity is at least the same (should be the same buffer)
	if cap(buf2) < capacity {
		t.Fatalf("Expected capacity at least %d, got %d", capacity, cap(buf2))
	}

	// Grow the buffer to expose the underlying memory
	buf2 = append(buf2, make([]byte, len(sensitiveData))...)

	// Write some new data to the buffer to expose the underlying memory
	newData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	copy(buf2, newData)

	// Verify that the sensitive data was zeroed out
	// The new data should be there, but no trace of the old data
	for i, b := range buf2[:len(newData)] {
		if b != newData[i] {
			t.Fatalf("Expected byte %d to be %d, got %d", i, newData[i], b)
		}
	}
}

func BenchmarkWithPool(b *testing.B) {
	pool := NewPool()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		buf = append(buf, []byte("benchmark test data")...)
		pool.Put(buf)
	}
}

func BenchmarkWithoutPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 0, 64)
		buf = append(buf, []byte("benchmark test data")...)
	}
}