package pubhash

import (
	"io"

	"github.com/minio/sha256-simd"
	"not.realy.lol/chk"
	"not.realy.lol/ec/schnorr"
	"not.realy.lol/errorf"
	"not.realy.lol/hex"
)

const Len = 8

type T struct{ val [Len]byte }

func (ph *T) FromPubkey(pk []byte) (err error) {
	if len(pk) != schnorr.PubKeyBytesLen {
		err = errorf.E(
			"invalid Pubkey length, got %d require %d",
			len(pk), schnorr.PubKeyBytesLen,
		)
		return
	}
	pkh := sha256.Sum256(pk)
	copy(ph.val[:], pkh[:Len])
	return
}

func (ph *T) FromPubkeyHex(pk string) (err error) {
	if len(pk) != schnorr.PubKeyBytesLen*2 {
		err = errorf.E(
			"invalid Pubkey length, got %d require %d",
			len(pk), schnorr.PubKeyBytesLen*2,
		)
		return
	}
	var pkb []byte
	if pkb, err = hex.Dec(pk); chk.E(err) {
		return
	}
	h := sha256.Sum256(pkb)
	copy(ph.val[:], h[:Len])
	return
}

func (ph *T) Bytes() (b []byte) { return ph.val[:] }

func (ph *T) MarshalWrite(w io.Writer) (err error) {
	_, err = w.Write(ph.val[:])
	return
}

func (ph *T) UnmarshalRead(r io.Reader) (err error) {
	copy(ph.val[:], ph.val[:Len])
	_, err = r.Read(ph.val[:])
	return
}
