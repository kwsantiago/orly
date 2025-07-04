// Package codecbuf provides a concurrent-safe bytes buffer pool for encoding
// data.
package codecbuf

import (
	"bytes"
	"sync"
)

// Pool is a concurrent-safe pool of bytes.Buffer objects.
type Pool struct {
	pool sync.Pool
}

// NewPool creates a new buffer pool.
func NewPool() *Pool {
	return &Pool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get returns a buffer from the pool or creates a new one if the pool is empty.
func (p *Pool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

// Put returns a buffer to the pool after zeroing its bytes for security and resetting it.
func (p *Pool) Put(buf *bytes.Buffer) {
	// Zero out the bytes for security
	data := buf.Bytes()
	for i := range data {
		data[i] = 0
	}
	buf.Reset()
	p.pool.Put(buf)
}

// DefaultPool is the default buffer pool for the application.
var DefaultPool = NewPool()

// Get returns a buffer from the default pool.
func Get() *bytes.Buffer {
	return DefaultPool.Get()
}

// Put returns a buffer to the default pool after zeroing its bytes for security.
func Put(buf *bytes.Buffer) {
	DefaultPool.Put(buf)
}
