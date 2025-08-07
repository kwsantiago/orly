//go:build !cgo

package btcec

import (
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/utils/chk"
)

func NewSecFromHex[V []byte | string](skh V) (sign signer.I, err error) {
	sk := make([]byte, len(skh)/2)
	if _, err = hex.DecBytes(sk, []byte(skh)); chk.E(err) {
		return
	}
	sign = &Signer{}
	if err = sign.InitSec(sk); chk.E(err) {
		return
	}
	return
}

func NewPubFromHex[V []byte | string](pkh V) (sign signer.I, err error) {
	pk := make([]byte, len(pkh)/2)
	if _, err = hex.DecBytes(pk, []byte(pkh)); chk.E(err) {
		return
	}
	sign = &Signer{}
	if err = sign.InitPub(pk); chk.E(err) {
		return
	}
	return
}

func HexToBin(hexStr string) (b []byte, err error) {
	b = make([]byte, len(hexStr)/2)
	if _, err = hex.DecBytes(b, []byte(hexStr)); chk.E(err) {
		return
	}
	return
}
