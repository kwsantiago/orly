package nwc_test

import (
	"encoding/json"
	"testing"
	"orly.dev/pkg/crypto/encryption"
	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/protocol/nwc"
)

func TestNWCConversationKey(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	walletPubkey := "816fd7f1d000ae81a3da251c91866fc47f4bcd6ce36921e6d46773c32f1d548b"
	
	uri := "nostr+walletconnect://" + walletPubkey + "?relay=wss://relay.getalby.com/v1&secret=" + secret
	
	parts, err := nwc.ParseConnectionURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	
	// Validate conversation key was generated
	convKey := parts.GetConversationKey()
	if len(convKey) == 0 {
		t.Fatal("conversation key should not be empty")
	}
	
	// Validate wallet public key
	walletKey := parts.GetWalletPublicKey()
	if len(walletKey) == 0 {
		t.Fatal("wallet public key should not be empty")
	}
	
	expected, err := hex.Dec(walletPubkey)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(walletKey) != len(expected) {
		t.Fatal("wallet public key length mismatch")
	}
	
	for i := range walletKey {
		if walletKey[i] != expected[i] {
			t.Fatal("wallet public key mismatch")
		}
	}
	
	t.Log("✅ Conversation key and wallet pubkey validation passed")
}

func TestNWCEncryptionDecryption(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	walletPubkey := "816fd7f1d000ae81a3da251c91866fc47f4bcd6ce36921e6d46773c32f1d548b"
	
	uri := "nostr+walletconnect://" + walletPubkey + "?relay=wss://relay.getalby.com/v1&secret=" + secret
	
	parts, err := nwc.ParseConnectionURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	
	convKey := parts.GetConversationKey()
	testMessage := `{"method":"get_info","params":null}`
	
	// Test encryption
	encrypted, err := encryption.Encrypt([]byte(testMessage), convKey)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	
	if len(encrypted) == 0 {
		t.Fatal("encrypted message should not be empty")
	}
	
	// Test decryption
	decrypted, err := encryption.Decrypt(encrypted, convKey)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	
	if string(decrypted) != testMessage {
		t.Fatalf("decrypted message mismatch: got %s, want %s", string(decrypted), testMessage)
	}
	
	t.Log("✅ NWC encryption/decryption cycle validated")
}

func TestNWCEventCreation(t *testing.T) {
	secretBytes, err := hex.Dec("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	
	clientKey := &p256k.Signer{}
	if err := clientKey.InitSec(secretBytes); err != nil {
		t.Fatal(err)
	}
	
	walletPubkey, err := hex.Dec("816fd7f1d000ae81a3da251c91866fc47f4bcd6ce36921e6d46773c32f1d548b")
	if err != nil {
		t.Fatal(err)
	}
	
	convKey, err := encryption.GenerateConversationKeyWithSigner(clientKey, walletPubkey)
	if err != nil {
		t.Fatal(err)
	}
	
	request := map[string]any{"method": "get_info"}
	reqBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	
	encrypted, err := encryption.Encrypt(reqBytes, convKey)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create NWC event
	ev := &event.E{
		Content:   encrypted,
		CreatedAt: timestamp.Now(),
		Kind:      kind.New(23194),
		Tags: tags.New(
			tag.New("encryption", "nip44_v2"),
			tag.New("p", hex.Enc(walletPubkey)),
		),
	}
	
	if err := ev.Sign(clientKey); err != nil {
		t.Fatalf("event signing failed: %v", err)
	}
	
	// Validate event structure
	if len(ev.Content) == 0 {
		t.Fatal("event content should not be empty")
	}
	
	if len(ev.ID) == 0 {
		t.Fatal("event should have ID after signing")
	}
	
	if len(ev.Sig) == 0 {
		t.Fatal("event should have signature after signing")
	}
	
	// Validate tags
	hasEncryption := false
	hasP := false
	for i := 0; i < ev.Tags.Len(); i++ {
		tag := ev.Tags.GetTagElement(i)
		if tag.Len() >= 2 {
			if tag.S(0) == "encryption" && tag.S(1) == "nip44_v2" {
				hasEncryption = true
			}
			if tag.S(0) == "p" && tag.S(1) == hex.Enc(walletPubkey) {
				hasP = true
			}
		}
	}
	
	if !hasEncryption {
		t.Fatal("event missing encryption tag")
	}
	
	if !hasP {
		t.Fatal("event missing p tag")
	}
	
	t.Log("✅ NWC event creation and signing validated")
}