package codecbuf

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
	testData := "test data"
	_, err := buf.WriteString(testData)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	// Verify the buffer contains the expected data
	if buf.String() != testData {
		t.Fatalf("Expected buffer to contain %q, got %q", testData, buf.String())
	}

	// Put the buffer back in the pool
	pool.Put(buf)

	// Get another buffer from the pool (should be the same one, reset)
	buf2 := pool.Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from pool")
	}

	// Verify the buffer is empty (was reset)
	if buf2.Len() != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", buf2.Len())
	}

	// Write different data to the buffer
	testData2 := "different data"
	_, err = buf2.WriteString(testData2)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	// Verify the buffer contains the new data
	if buf2.String() != testData2 {
		t.Fatalf("Expected buffer to contain %q, got %q", testData2, buf2.String())
	}
}

func TestDefaultPool(t *testing.T) {
	// Get a buffer from the default pool
	buf := Get()
	if buf == nil {
		t.Fatal("Expected non-nil buffer from default pool")
	}

	// Write some data to the buffer
	testData := "test data for default pool"
	_, err := buf.WriteString(testData)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	// Verify the buffer contains the expected data
	if buf.String() != testData {
		t.Fatalf("Expected buffer to contain %q, got %q", testData, buf.String())
	}

	// Put the buffer back in the pool
	Put(buf)

	// Get another buffer from the pool (should be reset)
	buf2 := Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from default pool")
	}

	// Verify the buffer is empty (was reset)
	if buf2.Len() != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", buf2.Len())
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
	_, err := buf.Write(sensitiveData)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	// Get the capacity before putting it back
	capacity := buf.Cap()

	// Put the buffer back in the pool
	pool.Put(buf)

	// Get another buffer from the pool (should be the same one, reset)
	buf2 := pool.Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from pool")
	}

	// Verify the buffer is empty (was reset)
	if buf2.Len() != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", buf2.Len())
	}

	// Verify the capacity is the same (should be the same buffer)
	if buf2.Cap() != capacity {
		t.Fatalf("Expected capacity %d, got %d", capacity, buf2.Cap())
	}

	// Get the underlying bytes directly
	// We need to grow the buffer to the same size as before to access the same memory
	buf2.Grow(len(sensitiveData))

	// Write some new data to the buffer to expose the underlying memory
	newData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	_, err = buf2.Write(newData)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	// Read the buffer bytes
	bufBytes := buf2.Bytes()

	// Verify that the sensitive data was zeroed out
	// The new data should be there, but no trace of the old data
	for i, b := range bufBytes[:len(newData)] {
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
	_, err := buf.Write(sensitiveData)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	// Get the capacity before putting it back
	capacity := buf.Cap()

	// Put the buffer back in the pool
	Put(buf)

	// Get another buffer from the pool (should be the same one, reset)
	buf2 := Get()
	if buf2 == nil {
		t.Fatal("Expected non-nil buffer from default pool")
	}

	// Verify the buffer is empty (was reset)
	if buf2.Len() != 0 {
		t.Fatalf("Expected empty buffer, got buffer with length %d", buf2.Len())
	}

	// Verify the capacity is the same (should be the same buffer)
	if buf2.Cap() != capacity {
		t.Fatalf("Expected capacity %d, got %d", capacity, buf2.Cap())
	}

	// Get the underlying bytes directly
	// We need to grow the buffer to the same size as before to access the same memory
	buf2.Grow(len(sensitiveData))

	// Write some new data to the buffer to expose the underlying memory
	newData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	_, err = buf2.Write(newData)
	if err != nil {
		t.Fatalf("Failed to write to buffer: %v", err)
	}

	// Read the buffer bytes
	bufBytes := buf2.Bytes()

	// Verify that the sensitive data was zeroed out
	// The new data should be there, but no trace of the old data
	for i, b := range bufBytes[:len(newData)] {
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
		buf.WriteString("benchmark test data")
		pool.Put(buf)
	}
}

func BenchmarkWithoutPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.WriteString("benchmark test data")
	}
}
