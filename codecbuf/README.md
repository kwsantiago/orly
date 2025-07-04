# Codecbuf - Concurrent-Safe Bytes Buffer Pool

This package provides a concurrent-safe pool of `bytes.Buffer` objects for encoding data. It helps reduce memory allocations and improve performance by reusing buffers instead of creating new ones for each operation.

## Usage

### Basic Usage

```go
// Get a buffer from the default pool
buf := codecbuf.Get()

// Use the buffer
buf.WriteString("Hello, World!")
// ... do more operations with the buffer ...

// Return the buffer to the pool when done
codecbuf.Put(buf)
```

### Using with defer

```go
func ProcessData() {
    // Get a buffer from the default pool
    buf := codecbuf.Get()
    
    // Return the buffer to the pool when the function exits
    defer codecbuf.Put(buf)
    
    // Use the buffer
    buf.WriteString("Hello, World!")
    // ... do more operations with the buffer ...
}
```

### Creating a Custom Pool

```go
// Create a new buffer pool
pool := codecbuf.NewPool()

// Get a buffer from the custom pool
buf := pool.Get()

// Use the buffer
buf.WriteString("Hello, World!")

// Return the buffer to the custom pool
pool.Put(buf)
```

## Performance

Using a buffer pool can significantly improve performance in applications that frequently create and use byte buffers, especially in high-throughput scenarios. The pool reduces garbage collection pressure by reusing buffers instead of allocating new ones.

## Thread Safety

The buffer pool is safe for concurrent use by multiple goroutines. However, individual buffers obtained from the pool should not be used concurrently by multiple goroutines without additional synchronization.