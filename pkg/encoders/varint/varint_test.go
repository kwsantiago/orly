package varint

import (
	"bytes"
	"math"
	"orly.dev/pkg/encoders/codecbuf"
	"orly.dev/pkg/utils/chk"
	"testing"

	"lukechampine.com/frand"
)

func TestEncode_Decode(t *testing.T) {
	var v uint64
	for range 10000000 {
		v = uint64(frand.Intn(math.MaxInt64))
		buf1 := codecbuf.Get()
		Encode(buf1, v)
		buf2 := bytes.NewBuffer(buf1.Bytes())
		u, err := Decode(buf2)
		if chk.E(err) {
			t.Fatal(err)
		}
		if u != v {
			t.Fatalf("expected %d got %d", v, u)
		}

	}
}
