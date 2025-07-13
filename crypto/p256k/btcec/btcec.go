// Package btcec implements the signer.I interface for signatures and ECDH with nostr.
package btcec

import (
	btcec2 "orly.dev/crypto/ec"
	schnorr2 "orly.dev/crypto/ec/schnorr"
	secp256k2 "orly.dev/crypto/ec/secp256k1"
	"orly.dev/interfaces/signer"
	"orly.dev/utils/chk"
	"orly.dev/utils/errorf"
)

// Signer is an implementation of signer.I that uses the btcec library.
type Signer struct {
	SecretKey *secp256k2.SecretKey
	PublicKey *secp256k2.PublicKey
	BTCECSec  *btcec2.SecretKey
	pkb, skb  []byte
}

var _ signer.I = &Signer{}

// Generate creates a new Signer.
func (s *Signer) Generate() (err error) {
	if s.SecretKey, err = btcec2.NewSecretKey(); chk.E(err) {
		return
	}
	s.skb = s.SecretKey.Serialize()
	s.BTCECSec, _ = btcec2.PrivKeyFromBytes(s.skb)
	s.PublicKey = s.SecretKey.PubKey()
	s.pkb = schnorr2.SerializePubKey(s.PublicKey)
	return
}

// InitSec initialises a Signer using raw secret key bytes.
func (s *Signer) InitSec(sec []byte) (err error) {
	if len(sec) != secp256k2.SecKeyBytesLen {
		err = errorf.E("sec key must be %d bytes", secp256k2.SecKeyBytesLen)
		return
	}
	s.SecretKey = secp256k2.SecKeyFromBytes(sec)
	s.PublicKey = s.SecretKey.PubKey()
	s.pkb = schnorr2.SerializePubKey(s.PublicKey)
	s.BTCECSec, _ = btcec2.PrivKeyFromBytes(s.skb)
	return
}

// InitPub initializes a signature verifier Signer from raw public key bytes.
func (s *Signer) InitPub(pub []byte) (err error) {
	if s.PublicKey, err = schnorr2.ParsePubKey(pub); chk.E(err) {
		return
	}
	s.pkb = pub
	return
}

// Sec returns the raw secret key bytes.
func (s *Signer) Sec() (b []byte) { return s.skb }

// Pub returns the raw BIP-340 schnorr public key bytes.
func (s *Signer) Pub() (b []byte) { return s.pkb }

// Sign a message with the Signer. Requires an initialised secret key.
func (s *Signer) Sign(msg []byte) (sig []byte, err error) {
	if s.SecretKey == nil {
		err = errorf.E("btcec: Signer not initialized")
		return
	}
	var si *schnorr2.Signature
	if si, err = schnorr2.Sign(s.SecretKey, msg); chk.E(err) {
		return
	}
	sig = si.Serialize()
	return
}

// Verify a message signature, only requires the public key is initialised.
func (s *Signer) Verify(msg, sig []byte) (valid bool, err error) {
	if s.PublicKey == nil {
		err = errorf.E("btcec: Pubkey not initialized")
		return
	}
	var si *schnorr2.Signature
	if si, err = schnorr2.ParseSignature(sig); chk.D(err) {
		err = errorf.E(
			"failed to parse signature:\n%d %s\n%v", len(sig),
			sig, err,
		)
		return
	}
	valid = si.Verify(msg, s.PublicKey)
	return
}

// Zero wipes the bytes of the secret key.
func (s *Signer) Zero() { s.SecretKey.Key.Zero() }

// ECDH creates a shared secret from a secret key and a provided public key bytes. It is advised
// to hash this result for security reasons.
func (s *Signer) ECDH(pubkeyBytes []byte) (secret []byte, err error) {
	var pub *secp256k2.PublicKey
	if pub, err = secp256k2.ParsePubKey(
		append(
			[]byte{0x02}, pubkeyBytes...,
		),
	); chk.E(err) {
		return
	}
	secret = btcec2.GenerateSharedSecret(s.BTCECSec, pub)
	return
}

// Keygen implements a key generator. Used for such things as vanity npub mining.
type Keygen struct {
	Signer
}

// Generate a new key pair. If the result is suitable, the embedded Signer can have its contents
// extracted.
func (k *Keygen) Generate() (pubBytes []byte, err error) {
	if k.Signer.SecretKey, err = btcec2.NewSecretKey(); chk.E(err) {
		return
	}
	k.Signer.PublicKey = k.SecretKey.PubKey()
	k.Signer.pkb = schnorr2.SerializePubKey(k.Signer.PublicKey)
	pubBytes = k.Signer.pkb
	return
}

// KeyPairBytes returns the raw bytes of the embedded Signer.
func (k *Keygen) KeyPairBytes() (secBytes, cmprPubBytes []byte) {
	return k.Signer.SecretKey.Serialize(), k.Signer.PublicKey.SerializeCompressed()
}
