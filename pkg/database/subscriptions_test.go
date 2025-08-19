package database

import (
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestSubscriptionLifecycle(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	d := &D{DB: db}
	pubkey := []byte("test_pubkey_32_bytes_long_enough")

	// First check should create trial
	active, err := d.IsSubscriptionActive(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Error("expected trial to be active")
	}

	// Verify trial was created
	sub, err := d.GetSubscription(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if sub == nil {
		t.Fatal("expected subscription to exist")
	}
	if sub.TrialEnd.IsZero() {
		t.Error("expected trial end to be set")
	}
	if !sub.PaidUntil.IsZero() {
		t.Error("expected paid until to be zero")
	}

	// Extend subscription
	err = d.ExtendSubscription(pubkey, 30)
	if err != nil {
		t.Fatal(err)
	}

	// Check subscription is still active
	active, err = d.IsSubscriptionActive(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Error("expected subscription to be active after extension")
	}

	// Verify paid until was set
	sub, err = d.GetSubscription(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if sub.PaidUntil.IsZero() {
		t.Error("expected paid until to be set after extension")
	}
}

func TestExtendSubscriptionEdgeCases(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	d := &D{DB: db}
	pubkey := []byte("test_pubkey_32_bytes_long_enough")

	// Test extending non-existent subscription
	err = d.ExtendSubscription(pubkey, 30)
	if err != nil {
		t.Fatal(err)
	}

	sub, err := d.GetSubscription(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if sub.PaidUntil.IsZero() {
		t.Error("expected paid until to be set")
	}

	// Test invalid days
	err = d.ExtendSubscription(pubkey, 0)
	if err == nil {
		t.Error("expected error for 0 days")
	}

	err = d.ExtendSubscription(pubkey, -1)
	if err == nil {
		t.Error("expected error for negative days")
	}
}

func TestGetNonExistentSubscription(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	d := &D{DB: db}
	pubkey := []byte("non_existent_pubkey_32_bytes_long")

	sub, err := d.GetSubscription(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if sub != nil {
		t.Error("expected nil for non-existent subscription")
	}
}
