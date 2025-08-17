package keys

import (
	"orly.dev/pkg/crypto/ec/bech32"
	"orly.dev/pkg/encoders/bech32encoding"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/utils"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
)

func DecodeNpubOrHex(v string) (pk []byte, err error) {
	var prf []byte
	var bits5 []byte
	if prf, bits5, err = bech32.DecodeNoLimit([]byte(v)); chk.D(err) {
		// try hex then
		if _, err = hex.DecBytes(pk, []byte(v)); chk.E(err) {
			log.W.F(
				"owner key %s is neither bech32 npub nor hex",
				v,
			)
			return
		}
		// it was hex, return
		return
	}
	if !utils.FastEqual(prf, bech32encoding.NpubHRP) {
		log.W.F(
			"owner key %s is neither bech32 npub nor hex",
			v,
		)
		return
	}
	if pk, err = bech32.ConvertBits(bits5, 5, 8, false); chk.E(err) {
		return
	}
	return
}

func DecodeNsecOrHex(v string) (sk []byte, err error) {
	var prf []byte
	var bits5 []byte
	if prf, bits5, err = bech32.DecodeNoLimit([]byte(v)); chk.D(err) {
		// try hex then
		if _, err = hex.DecBytes(sk, []byte(v)); chk.E(err) {
			log.W.F(
				"owner key %s is neither bech32 nsec nor hex",
				v,
			)
			return
		}
		return
	}
	if !utils.FastEqual(prf, bech32encoding.NsecHRP) {
		log.W.F(
			"owner key %s is neither bech32 nsec nor hex",
			v,
		)
		return
	}
	if sk, err = bech32.ConvertBits(bits5, 5, 8, false); chk.E(err) {
		return
	}
	return
}
