package types

import (
	"io"
	"not.realy.lol/chk"
)

const LetterLen = 1

type Letter struct {
	val byte
}

func (p *Letter) Set(lb byte) { p.val = lb }

func (p *Letter) Letter() byte { return p.val }

func (p *Letter) MarshalWrite(w io.Writer) (err error) {
	_, err = w.Write([]byte{p.val})
	return
}

func (p *Letter) UnmarshalRead(r io.Reader) (err error) {
	val := make([]byte, 1)
	if _, err = r.Read(val); chk.E(err) {
		return
	}
	p.val = val[0]
	return
}
