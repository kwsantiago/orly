package types

import (
	"io"

	"github.com/minio/sha256-simd"
)

const IdentLen = 8

type Ident struct{ val [IdentLen]byte }

func (i *Ident) FromIdent(id []byte) (err error) {
	idh := sha256.Sum256(id)
	copy(i.val[:], idh[:IdentLen])
	return
}

func (i *Ident) Bytes() (b []byte) { return i.val[:] }

func (i *Ident) MarshalWrite(w io.Writer) (err error) {
	_, err = w.Write(i.val[:])
	return
}

func (i *Ident) UnmarshalRead(r io.Reader) (err error) {

	copy(i.val[:], i.val[:IdentLen])
	_, err = r.Read(i.val[:])
	return
}
