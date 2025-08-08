//go:build !cgo

// Package btcec implements the signer.I interface for signatures and ECDH with nostr.
package btcec

import (
	btcec3 "orly.dev/pkg/crypto/ec"
	"orly.dev/pkg/crypto/ec/schnorr"
	"orly.dev/pkg/crypto/ec/secp256k1"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/errorf"
)

// Signer is an implementation of signer.I that uses the btcec library.
type Signer struct {
	SecretKey *secp256k1.SecretKey
	PublicKey *secp256k1.PublicKey
	BTCECSec  *btcec3.SecretKey
	pkb, skb  []byte
}

var _ signer.I = &Signer{}

// Generate creates a new Signer.
func (s *Signer) Generate() (err error) {
	if s.SecretKey, err = btcec3.NewSecretKey(); chk.E(err) {
		return
	}
	s.skb = s.SecretKey.Serialize()
	s.BTCECSec, _ = btcec3.PrivKeyFromBytes(s.skb)
	s.PublicKey = s.SecretKey.PubKey()
	s.pkb = schnorr.SerializePubKey(s.PublicKey)
	return
}

// InitSec initialises a Signer using raw secret key bytes.
func (s *Signer) InitSec(sec []byte) (err error) {
	if len(sec) != secp256k1.SecKeyBytesLen {
		err = errorf.E("sec key must be %d bytes", secp256k1.SecKeyBytesLen)
		return
	}
	s.skb = sec
	s.SecretKey = secp256k1.SecKeyFromBytes(sec)
	s.PublicKey = s.SecretKey.PubKey()
	s.pkb = schnorr.SerializePubKey(s.PublicKey)
	s.BTCECSec, _ = btcec3.PrivKeyFromBytes(s.skb)
	return
}

// InitPub initializes a signature verifier Signer from raw public key bytes.
func (s *Signer) InitPub(pub []byte) (err error) {
	if s.PublicKey, err = schnorr.ParsePubKey(pub); chk.E(err) {
		return
	}
	s.pkb = pub
	return
}

// Sec returns the raw secret key bytes.
func (s *Signer) Sec() (b []byte) {
	if s == nil {
		return nil
	}
	return s.skb
}

// Pub returns the raw BIP-340 schnorr public key bytes.
func (s *Signer) Pub() (b []byte) {
	if s == nil {
		return nil
	}
	return s.pkb
}

// Sign a message with the Signer. Requires an initialised secret key.
func (s *Signer) Sign(msg []byte) (sig []byte, err error) {
	if s.SecretKey == nil {
		err = errorf.E("btcec: Signer not initialized")
		return
	}
	var si *schnorr.Signature
	if si, err = schnorr.Sign(s.SecretKey, msg); chk.E(err) {
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

	// First try to verify using the schnorr package
	var si *schnorr.Signature
	if si, err = schnorr.ParseSignature(sig); err == nil {
		valid = si.Verify(msg, s.PublicKey)
		return
	}

	// If parsing the signature failed, log it at debug level
	chk.D(err)

	// If the signature is exactly 64 bytes, try to verify it directly
	// This is to handle signatures created by p256k.Signer which uses libsecp256k1
	if len(sig) == schnorr.SignatureSize {
		// Create a new signature with the raw bytes
		var r secp256k1.FieldVal
		var sScalar secp256k1.ModNScalar

		// Split the signature into r and s components
		if overflow := r.SetByteSlice(sig[0:32]); !overflow {
			sScalar.SetByteSlice(sig[32:64])

			// Create a new signature and verify it
			newSig := schnorr.NewSignature(&r, &sScalar)
			valid = newSig.Verify(msg, s.PublicKey)
			return
		}
	}

	// If all verification methods failed, return an error
	err = errorf.E(
		"failed to verify signature:\n%d %s", len(sig), sig,
	)
	return
}

// Zero wipes the bytes of the secret key.
func (s *Signer) Zero() { s.SecretKey.Key.Zero() }

// ECDH creates a shared secret from a secret key and a provided public key bytes. It is advised
// to hash this result for security reasons.
func (s *Signer) ECDH(pubkeyBytes []byte) (secret []byte, err error) {
	var pub *secp256k1.PublicKey
	if pub, err = secp256k1.ParsePubKey(
		append(
			[]byte{0x02}, pubkeyBytes...,
		),
	); chk.E(err) {
		return
	}
	secret = btcec3.GenerateSharedSecret(s.BTCECSec, pub)
	return
}

// Keygen implements a key generator. Used for such things as vanity npub mining.
type Keygen struct {
	Signer
}

// Generate a new key pair. If the result is suitable, the embedded Signer can have its contents
// extracted.
func (k *Keygen) Generate() (pubBytes []byte, err error) {
	if k.Signer.SecretKey, err = btcec3.NewSecretKey(); chk.E(err) {
		return
	}
	k.Signer.PublicKey = k.SecretKey.PubKey()
	k.Signer.pkb = schnorr.SerializePubKey(k.Signer.PublicKey)
	pubBytes = k.Signer.pkb
	return
}

// KeyPairBytes returns the raw bytes of the embedded Signer.
func (k *Keygen) KeyPairBytes() (secBytes, cmprPubBytes []byte) {
	return k.Signer.SecretKey.Serialize(), k.Signer.PublicKey.SerializeCompressed()
}
