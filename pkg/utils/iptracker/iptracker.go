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
	mu             sync.RWMutex
	failedAttempts map[string]int
	blockedUntil   map[string]time.Time
	offenseCount   map[string]int           // Tracks the number of times an IP has been blocked
	blockDurations map[string]time.Duration // Stores the current block duration for each IP
}

// NewIPTracker creates a new IPTracker instance.
func NewIPTracker() *IPTracker {
	return &IPTracker{
		failedAttempts: make(map[string]int),
		blockedUntil:   make(map[string]time.Time),
		offenseCount:   make(map[string]int),
		blockDurations: make(map[string]time.Duration),
	}
}

// RecordFailedAttempt records a failed authentication attempt for the given IP address.
// If the number of failed attempts exceeds the threshold, the IP is blocked.
// For repeat offenders, the block duration doubles with each offense.
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
		// Increment the offense count
		t.offenseCount[ip]++

		// Calculate block duration based on offense count
		// First offense: 10 minutes, then doubles for each subsequent offense
		duration := BlockDuration
		if t.offenseCount[ip] > 1 {
			// For repeat offenses, double the duration for each previous offense
			// 10 min, then 20, then 40, then 80, etc.
			for i := 1; i < t.offenseCount[ip]; i++ {
				duration *= 2
			}
		}

		// Store the calculated duration
		t.blockDurations[ip] = duration

		// Set the block time
		t.blockedUntil[ip] = time.Now().Add(duration)
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
// Blocks persist until authentication, even after the block duration has passed.
func (t *IPTracker) isBlockedNoLock(ip string) bool {
	_, exists := t.blockedUntil[ip]
	if !exists {
		return false
	}

	// IP is blocked until authenticated, regardless of time elapsed
	return true
}

// HasBlockDurationPassed checks if the block duration for an IP has passed,
// even though the IP remains blocked until authentication.
// This is useful for display purposes.
func (t *IPTracker) HasBlockDurationPassed(ip string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	blockedUntil, exists := t.blockedUntil[ip]
	if !exists {
		return false
	}

	return time.Now().After(blockedUntil)
}

// GetBlockedUntil returns the time until which the given IP address is blocked.
// If the IP is not blocked, it returns the zero time.
// Note: With the new blocking behavior, an IP remains blocked even after this time
// until it successfully authenticates.
func (t *IPTracker) GetBlockedUntil(ip string) time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	blockedUntil, exists := t.blockedUntil[ip]
	if !exists {
		return time.Time{}
	}

	return blockedUntil
}

// GetBlockDuration returns the current block duration for the given IP address.
// This is useful for displaying how long the IP would have been blocked before
// requiring authentication.
func (t *IPTracker) GetBlockDuration(ip string) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	duration, exists := t.blockDurations[ip]
	if !exists {
		return 0
	}

	return duration
}

// Authenticate records a successful authentication for an IP address.
// If the IP was blocked, it removes the block but preserves the offense count.
// This allows the IP to access the system again, but if it offends in the future,
// the penalty will still be doubled based on past offenses.
func (t *IPTracker) Authenticate(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Remove the block but keep the offense count
	delete(t.failedAttempts, ip)
	delete(t.blockedUntil, ip)
	// Note: We intentionally don't delete from offenseCount or blockDurations
	// so that repeat offenses can be tracked
}

// Reset completely resets all tracking for the given IP address.
// This is different from Authenticate as it also resets the offense count.
func (t *IPTracker) Reset(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.failedAttempts, ip)
	delete(t.blockedUntil, ip)
	delete(t.offenseCount, ip)
	delete(t.blockDurations, ip)
}

// Global instance of IPTracker for use across the application
var Global = NewIPTracker()
