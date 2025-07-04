package idhash

import (
	"encoding/base64"
	"io"

	"github.com/minio/sha256-simd"

	"not.realy.lol/chk"
	"not.realy.lol/errorf"
	"not.realy.lol/hex"
)

const Len = 8

type T struct{ val []byte }

func New() (i *T) { return &T{make([]byte, Len)} }

func (i *T) FromId(id []byte) (err error) {
	if len(id) != sha256.Size {
		err = errorf.E(
			"invalid Id length, got %d require %d", len(id), sha256.Size,
		)
		return
	}
	idh := sha256.Sum256(id)
	i.val = idh[:Len]
	return
}

func (i *T) FromIdBase64(idb64 string) (err error) {
	// Decode the base64 string
	decoded, err := base64.RawURLEncoding.DecodeString(idb64)
	if chk.E(err) {
		return
	}

	// Check if the decoded ID has the correct length
	if len(decoded) != sha256.Size {
		err = errorf.E(
			"invalid Id length, got %d require %d", len(decoded), sha256.Size,
		)
		return
	}

	// Hash the decoded ID and take the first Len bytes
	idh := sha256.Sum256(decoded)
	i.val = idh[:Len]
	return
}

func (i *T) FromIdHex(idh string) (err error) {
	var id []byte
	if id, err = hex.Dec(idh); chk.E(err) {
		return
	}
	if len(id) != sha256.Size {
		err = errorf.E(
			"invalid Id length, got %d require %d", len(id), sha256.Size,
		)
		return
	}
	h := sha256.Sum256(id)
	i.val = h[:Len]
	return

}

func (i *T) Bytes() (b []byte) { return i.val }

func (i *T) MarshalWrite(w io.Writer) (err error) {
	_, err = w.Write(i.val)
	return
}

func (i *T) UnmarshalRead(r io.Reader) (err error) {
	if len(i.val) < Len {
		i.val = make([]byte, Len)
	} else {
		i.val = i.val[:Len]
	}
	_, err = r.Read(i.val)
	return
}
