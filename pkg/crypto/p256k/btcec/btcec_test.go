//go:build !cgo

package btcec_test

import (
	"bufio"
	"bytes"
	"orly.dev/pkg/utils"
	"testing"
	"time"

	"orly.dev/pkg/crypto/ec/schnorr"
	"orly.dev/pkg/crypto/p256k/btcec"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/event/examples"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
)

func TestSigner_Generate(t *testing.T) {
	for _ = range 100 {
		var err error
		signer := &btcec.Signer{}
		var skb []byte
		if err = signer.Generate(); chk.E(err) {
			t.Fatal(err)
		}
		skb = signer.Sec()
		if err = signer.InitSec(skb); chk.E(err) {
			t.Fatal(err)
		}
	}
}

// func TestBTCECSignerVerify(t *testing.T) {
// 	evs := make([]*event.E, 0, 10000)
// 	scanner := bufio.NewScanner(bytes.NewBuffer(examples.Cache))
// 	buf := make([]byte, 1_000_000)
// 	scanner.Buffer(buf, len(buf))
// 	var err error
//
// 	// Create both btcec and p256k signers
// 	btcecSigner := &btcec.Signer{}
// 	p256kSigner := &p256k.Signer{}
//
// 	for scanner.Scan() {
// 		var valid bool
// 		b := scanner.Bytes()
// 		ev := event.New()
// 		if _, err = ev.Unmarshal(b); chk.E(err) {
// 			t.Errorf("failed to marshal\n%s", b)
// 		} else {
// 			// We know ev.Verify() works, so we'll use it as a reference
// 			if valid, err = ev.Verify(); chk.E(err) || !valid {
// 				t.Errorf("invalid signature\n%s", b)
// 				continue
// 			}
// 		}
//
// 		// Get the ID from the event
// 		storedID := ev.ID
// 		calculatedID := ev.GetIDBytes()
//
// 		// Check if the stored ID matches the calculated ID
// 		if !utils.FastEqual(storedID, calculatedID) {
// 			log.D.Ln("Event ID mismatch: stored ID doesn't match calculated ID")
// 			// Use the calculated ID for verification as ev.Verify() would do
// 			ev.ID = calculatedID
// 		}
//
// 		if len(ev.ID) != sha256.Size {
// 			t.Errorf("id should be 32 bytes, got %d", len(ev.ID))
// 			continue
// 		}
//
// 		// Initialize both signers with the same public key
// 		if err = btcecSigner.InitPub(ev.Pubkey); chk.E(err) {
// 			t.Errorf("failed to init btcec pub key: %s\n%0x", err, b)
// 		}
// 		if err = p256kSigner.InitPub(ev.Pubkey); chk.E(err) {
// 			t.Errorf("failed to init p256k pub key: %s\n%0x", err, b)
// 		}
//
// 		// First try to verify with btcec.Signer
// 		if valid, err = btcecSigner.Verify(ev.ID, ev.Sig); err == nil && valid {
// 			// If btcec.Signer verification succeeds, great!
// 			log.D.Ln("btcec.Signer verification succeeded")
// 		} else {
// 			// If btcec.Signer verification fails, try with p256k.Signer
// 			// Use chk.T(err) like ev.Verify() does
// 			if valid, err = p256kSigner.Verify(ev.ID, ev.Sig); chk.T(err) {
// 				// If there's an error, log it but don't fail the test
// 				log.D.Ln("p256k.Signer verification error:", err)
// 			} else if !valid {
// 				// Only fail the test if both verifications fail
// 				t.Errorf(
// 					"invalid signature for pub %0x %0x %0x", ev.Pubkey, ev.ID,
// 					ev.Sig,
// 				)
// 			} else {
// 				log.D.Ln("p256k.Signer verification succeeded where btcec.Signer failed")
// 			}
// 		}
//
// 		evs = append(evs, ev)
// 	}
// }

func TestBTCECSignerSign(t *testing.T) {
	evs := make([]*event.E, 0, 10000)
	scanner := bufio.NewScanner(bytes.NewBuffer(examples.Cache))
	buf := make([]byte, 1_000_000)
	scanner.Buffer(buf, len(buf))
	var err error
	signer := &btcec.Signer{}
	var skb []byte
	if err = signer.Generate(); chk.E(err) {
		t.Fatal(err)
	}
	skb = signer.Sec()
	if err = signer.InitSec(skb); chk.E(err) {
		t.Fatal(err)
	}
	verifier := &btcec.Signer{}
	pkb := signer.Pub()
	if err = verifier.InitPub(pkb); chk.E(err) {
		t.Fatal(err)
	}
	counter := 0
	for scanner.Scan() {
		counter++
		if counter > 1000 {
			break
		}
		b := scanner.Bytes()
		ev := event.New()
		if _, err = ev.Unmarshal(b); chk.E(err) {
			t.Errorf("failed to marshal\n%s", b)
		}
		evs = append(evs, ev)
	}
	var valid bool
	sig := make([]byte, schnorr.SignatureSize)
	for _, ev := range evs {
		ev.Pubkey = pkb
		id := ev.GetIDBytes()
		if sig, err = signer.Sign(id); chk.E(err) {
			t.Errorf("failed to sign: %s\n%0x", err, id)
		}
		if valid, err = verifier.Verify(id, sig); chk.E(err) {
			t.Errorf("failed to verify: %s\n%0x", err, id)
		}
		if !valid {
			t.Errorf("invalid signature")
		}
	}
	signer.Zero()
}

func TestBTCECECDH(t *testing.T) {
	n := time.Now()
	var err error
	var counter int
	const total = 50
	for _ = range total {
		s1 := new(btcec.Signer)
		if err = s1.Generate(); chk.E(err) {
			t.Fatal(err)
		}
		s2 := new(btcec.Signer)
		if err = s2.Generate(); chk.E(err) {
			t.Fatal(err)
		}
		for _ = range total {
			var secret1, secret2 []byte
			if secret1, err = s1.ECDH(s2.Pub()); chk.E(err) {
				t.Fatal(err)
			}
			if secret2, err = s2.ECDH(s1.Pub()); chk.E(err) {
				t.Fatal(err)
			}
			if !utils.FastEqual(secret1, secret2) {
				counter++
				t.Errorf(
					"ECDH generation failed to work in both directions, %x %x",
					secret1,
					secret2,
				)
			}
		}
	}
	a := time.Now()
	duration := a.Sub(n)
	log.I.Ln(
		"errors", counter, "total", total, "time", duration, "time/op",
		int(duration/total),
		"ops/sec", int(time.Second)/int(duration/total),
	)
}
