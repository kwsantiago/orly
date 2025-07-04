package fullid

import (
	"io"

	"github.com/minio/sha256-simd"

	"not.realy.lol/errorf"
)

const Len = sha256.Size

type T struct {
	val [Len]byte
}

func (fi *T) FromId(id []byte) (err error) {
	if len(id) != Len {
		err = errorf.E("invalid Id length, got %d require %d", len(id), Len)
		return
	}
	copy(fi.val[:], id)
	return
}
func (fi *T) Bytes() (b []byte) { return fi.val[:] }

func (fi *T) MarshalWrite(w io.Writer) (err error) {
	_, err = w.Write(fi.val[:])
	return
}

func (fi *T) UnmarshalRead(r io.Reader) (err error) {
	copy(fi.val[:], fi.val[:Len])
	_, err = r.Read(fi.val[:])
	return
}
