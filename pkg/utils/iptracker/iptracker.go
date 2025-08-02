// Package iptracker provides functionality to track and block IP addresses
// based on failed authentication attempts.
package iptracker

import (
	"sync"
	"time"
)

const (
	// BlockDuration is the duration for which an IP will be blocked after
	// exceeding the maximum number of failed attempts.
	BlockDuration = 10 * time.Minute
)

// IPTracker tracks failed authentication attempts by IP address and provides
// functionality to block IPs that exceed a threshold.
type IPTracker struct {
	mu            sync.RWMutex
	failedAttempts map[string]int
	blockedUntil   map[string]time.Time
}

// NewIPTracker creates a new IPTracker instance.
func NewIPTracker() *IPTracker {
	return &IPTracker{
		failedAttempts: make(map[string]int),
		blockedUntil:   make(map[string]time.Time),
	}
}

// RecordFailedAttempt records a failed authentication attempt for the given IP address.
// If the number of failed attempts exceeds the threshold, the IP is blocked for the
// configured duration.
// Returns true if the IP is now blocked, false otherwise.
func (t *IPTracker) RecordFailedAttempt(ip string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if the IP is already blocked
	if t.isBlockedNoLock(ip) {
		return true
	}

	// Increment the failed attempts counter
	t.failedAttempts[ip]++

	// If the number of failed attempts exceeds the threshold, block the IP
	if t.failedAttempts[ip] >= 3 { // Threshold of 3 failed attempts
		t.blockedUntil[ip] = time.Now().Add(BlockDuration)
		return true
	}

	return false
}

// IsBlocked checks if the given IP address is currently blocked.
func (t *IPTracker) IsBlocked(ip string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isBlockedNoLock(ip)
}

// isBlockedNoLock is a helper method that checks if an IP is blocked without
// acquiring the lock. It should only be called when the lock is already held.
func (t *IPTracker) isBlockedNoLock(ip string) bool {
	blockedUntil, exists := t.blockedUntil[ip]
	if !exists {
		return false
	}

	// If the block has expired, remove it and return false
	if time.Now().After(blockedUntil) {
		delete(t.blockedUntil, ip)
		delete(t.failedAttempts, ip)
		return false
	}

	return true
}

// GetBlockedUntil returns the time until which the given IP address is blocked.
// If the IP is not blocked, it returns the zero time.
func (t *IPTracker) GetBlockedUntil(ip string) time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	blockedUntil, exists := t.blockedUntil[ip]
	if !exists || time.Now().After(blockedUntil) {
		return time.Time{}
	}

	return blockedUntil
}

// Reset resets the failed attempts counter for the given IP address.
func (t *IPTracker) Reset(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.failedAttempts, ip)
	delete(t.blockedUntil, ip)
}

// Global instance of IPTracker for use across the application
var Global = NewIPTracker()