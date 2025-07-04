package identhash

import (
	"io"

	"github.com/minio/sha256-simd"
)

const Len = 8

type T struct{ val [Len]byte }

func (i *T) FromIdent(id []byte) (err error) {
	idh := sha256.Sum256(id)
	copy(i.val[:], idh[:Len])
	return
}

func (i *T) Bytes() (b []byte) { return i.val[:] }

func (i *T) MarshalWrite(w io.Writer) (err error) {
	_, err = w.Write(i.val[:])
	return
}

func (i *T) UnmarshalRead(r io.Reader) (err error) {

	copy(i.val[:], i.val[:Len])
	_, err = r.Read(i.val[:])
	return
}
