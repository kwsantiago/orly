package relay

import (
	"testing"

	"github.com/dgraph-io/badger/v4"
	"orly.dev/pkg/app/config"
	"orly.dev/pkg/database"
)

func TestSubscriptionTrialActivation(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	d := &database.D{DB: db}
	pubkey := make([]byte, 32)

	// Test direct database calls
	active, err := d.IsSubscriptionActive(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("trial should be activated on first check")
	}

	// Verify subscription was created
	sub, err := d.GetSubscription(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if sub == nil {
		t.Fatal("subscription should exist")
	}
	if sub.TrialEnd.IsZero() {
		t.Error("trial end should be set")
	}
}

func TestSubscriptionExtension(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	d := &database.D{DB: db}
	pubkey := make([]byte, 32)

	// Create subscription and extend it
	err = d.ExtendSubscription(pubkey, 30)
	if err != nil {
		t.Fatal(err)
	}

	// Check it's active
	active, err := d.IsSubscriptionActive(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Error("subscription should be active after extension")
	}

	// Verify paid until is set
	sub, err := d.GetSubscription(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if sub.PaidUntil.IsZero() {
		t.Error("paid until should be set")
	}
}

func TestConfigValidation(t *testing.T) {
	// Test default values
	cfg := &config.C{}
	if cfg.SubscriptionEnabled {
		t.Error("subscription should be disabled by default")
	}
	if cfg.MonthlyPriceSats != 0 {
		t.Error("monthly price should be 0 by default before config load")
	}
}

func TestPaymentProcessingSimple(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	d := &database.D{DB: db}

	// Test payment recording
	pubkey := make([]byte, 32)
	err = d.RecordPayment(pubkey, 6000, "test_invoice", "test_preimage")
	if err != nil {
		t.Fatal(err)
	}

	// Test payment history retrieval
	payments, err := d.GetPaymentHistory(pubkey)
	if err != nil {
		t.Fatal(err)
	}
	if len(payments) != 1 {
		t.Errorf("expected 1 payment, got %d", len(payments))
	}
}
