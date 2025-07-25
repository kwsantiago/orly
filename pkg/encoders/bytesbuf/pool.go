// Package bytesbuf provides a concurrent-safe []byte buffer pool for encoding
// data.
package bytesbuf

import (
	"orly.dev/pkg/utils/units"
	"sync"
)

// Pool is a concurrent-safe pool of []byte objects.
type Pool struct {
	pool sync.Pool
}

// NewPool creates a new buffer pool.
func NewPool() *Pool {
	return &Pool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, units.Mb) // Initial capacity of 64 bytes
			},
		},
	}
}

// Get returns a buffer from the pool or creates a new one if the pool is empty.
func (p *Pool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a buffer to the pool after zeroing its bytes for security and resetting it.
func (p *Pool) Put(buf []byte) {
	// Zero out the bytes for security
	for i := range buf {
		buf[i] = 0
	}
	// Reset the slice length to 0 while preserving capacity
	p.pool.Put(buf[:0])
}

// DefaultPool is the default buffer pool for the application.
var DefaultPool = NewPool()

// Get returns a buffer from the default pool.
func Get() []byte {
	return DefaultPool.Get()
}

// Put returns a buffer to the default pool after zeroing its bytes for security.
func Put(buf []byte) {
	DefaultPool.Put(buf)
}
