package relay

import (
	"orly.dev/pkg/utils"
	"testing"
)

func TestLists_OwnersPubkeys(t *testing.T) {
	// Create a new Lists instance
	l := &Lists{}

	// Test with empty list
	pks := l.OwnersPubkeys()
	if len(pks) != 0 {
		t.Errorf("Expected empty list, got %d items", len(pks))
	}

	// Test with some pubkeys
	testPubkeys := [][]byte{
		[]byte("pubkey1"),
		[]byte("pubkey2"),
		[]byte("pubkey3"),
	}

	l.SetOwnersPubkeys(testPubkeys)

	// Verify length
	if l.LenOwnersPubkeys() != len(testPubkeys) {
		t.Errorf(
			"Expected length %d, got %d", len(testPubkeys),
			l.LenOwnersPubkeys(),
		)
	}

	// Verify content
	pks = l.OwnersPubkeys()
	if len(pks) != len(testPubkeys) {
		t.Errorf("Expected %d pubkeys, got %d", len(testPubkeys), len(pks))
	}

	// Verify each pubkey
	for i, pk := range pks {
		if !utils.FastEqual(pk, testPubkeys[i]) {
			t.Errorf(
				"Pubkey at index %d doesn't match: expected %s, got %s",
				i, testPubkeys[i], pk,
			)
		}
	}

	// Verify that the returned slice is a copy, not a reference
	pks[0] = []byte("modified")
	newPks := l.OwnersPubkeys()
	if utils.FastEqual(pks[0], newPks[0]) {
		t.Error("Returned slice should be a copy, not a reference")
	}
}

func TestLists_OwnersFollowed(t *testing.T) {
	// Create a new Lists instance
	l := &Lists{}

	// Test with empty list
	followed := l.OwnersFollowed()
	if len(followed) != 0 {
		t.Errorf("Expected empty list, got %d items", len(followed))
	}

	// Test with some pubkeys
	testPubkeys := [][]byte{
		[]byte("followed1"),
		[]byte("followed2"),
		[]byte("followed3"),
	}

	l.SetOwnersFollowed(testPubkeys)

	// Verify length
	if l.LenOwnersFollowed() != len(testPubkeys) {
		t.Errorf(
			"Expected length %d, got %d", len(testPubkeys),
			l.LenOwnersFollowed(),
		)
	}

	// Verify content
	followed = l.OwnersFollowed()
	if len(followed) != len(testPubkeys) {
		t.Errorf(
			"Expected %d followed, got %d", len(testPubkeys), len(followed),
		)
	}

	// Verify each pubkey
	for i, pk := range followed {
		if !utils.FastEqual(pk, testPubkeys[i]) {
			t.Errorf(
				"Followed at index %d doesn't match: expected %s, got %s",
				i, testPubkeys[i], pk,
			)
		}
	}
}

func TestLists_FollowedFollows(t *testing.T) {
	// Create a new Lists instance
	l := &Lists{}

	// Test with empty list
	follows := l.FollowedFollows()
	if len(follows) != 0 {
		t.Errorf("Expected empty list, got %d items", len(follows))
	}

	// Test with some pubkeys
	testPubkeys := [][]byte{
		[]byte("follow1"),
		[]byte("follow2"),
		[]byte("follow3"),
	}

	l.SetFollowedFollows(testPubkeys)

	// Verify length
	if l.LenFollowedFollows() != len(testPubkeys) {
		t.Errorf(
			"Expected length %d, got %d", len(testPubkeys),
			l.LenFollowedFollows(),
		)
	}

	// Verify content
	follows = l.FollowedFollows()
	if len(follows) != len(testPubkeys) {
		t.Errorf("Expected %d follows, got %d", len(testPubkeys), len(follows))
	}

	// Verify each pubkey
	for i, pk := range follows {
		if !utils.FastEqual(pk, testPubkeys[i]) {
			t.Errorf(
				"Follow at index %d doesn't match: expected %s, got %s",
				i, testPubkeys[i], pk,
			)
		}
	}
}

func TestLists_OwnersMuted(t *testing.T) {
	// Create a new Lists instance
	l := &Lists{}

	// Test with empty list
	muted := l.OwnersMuted()
	if len(muted) != 0 {
		t.Errorf("Expected empty list, got %d items", len(muted))
	}

	// Test with some pubkeys
	testPubkeys := [][]byte{
		[]byte("muted1"),
		[]byte("muted2"),
		[]byte("muted3"),
	}

	l.SetOwnersMuted(testPubkeys)

	// Verify length
	if l.LenOwnersMuted() != len(testPubkeys) {
		t.Errorf(
			"Expected length %d, got %d", len(testPubkeys), l.LenOwnersMuted(),
		)
	}

	// Verify content
	muted = l.OwnersMuted()
	if len(muted) != len(testPubkeys) {
		t.Errorf("Expected %d muted, got %d", len(testPubkeys), len(muted))
	}

	// Verify each pubkey
	for i, pk := range muted {
		if !utils.FastEqual(pk, testPubkeys[i]) {
			t.Errorf(
				"Muted at index %d doesn't match: expected %s, got %s",
				i, testPubkeys[i], pk,
			)
		}
	}
}

func TestLists_ConcurrentAccess(t *testing.T) {
	// Create a new Lists instance
	l := &Lists{}

	// Test concurrent access to the lists
	done := make(chan bool)

	// Concurrent reads and writes
	go func() {
		for i := 0; i < 100; i++ {
			l.SetOwnersPubkeys([][]byte{[]byte("pubkey1"), []byte("pubkey2")})
			l.OwnersPubkeys()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			l.SetOwnersFollowed(
				[][]byte{
					[]byte("followed1"), []byte("followed2"),
				},
			)
			l.OwnersFollowed()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			l.SetFollowedFollows([][]byte{[]byte("follow1"), []byte("follow2")})
			l.FollowedFollows()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			l.SetOwnersMuted([][]byte{[]byte("muted1"), []byte("muted2")})
			l.OwnersMuted()
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	// If we got here without deadlocks or panics, the test passes
}
