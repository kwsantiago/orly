package event

import (
	"bufio"
	"bytes"
	"orly.dev/encoders/codecbuf"
	"orly.dev/utils/chk"
	"testing"
	"time"

	"orly.dev/encoders/event/examples"
)

func TestTMarshalBinary_UnmarshalBinary(t *testing.T) {
	scanner := bufio.NewScanner(bytes.NewBuffer(examples.Cache))
	scanner.Buffer(make([]byte, 0, 1_000_000_000), 1_000_000_000)
	var rem, out []byte
	var err error
	buf := codecbuf.Get()
	ea, eb := New(), New()
	now := time.Now()
	var counter int
	for scanner.Scan() {
		chk.E(scanner.Err())
		b := scanner.Bytes()
		// log.I.F("%s", b)
		c := make([]byte, 0, len(b))
		c = append(c, b...)
		if rem, err = ea.Unmarshal(c); chk.E(err) {
			t.Fatal(err)
		}
		// log.I.F("len %d\n%s\n", len(b), ea.SerializeIndented())
		if len(rem) != 0 {
			t.Fatalf(
				"some of input remaining after marshal/unmarshal: '%s'",
				rem,
			)
		}
		ea.MarshalBinary(buf)
		// log.I.S(buf.Bytes())
		buf2 := bytes.NewBuffer(buf.Bytes())
		if err = eb.UnmarshalBinary(buf2); chk.E(err) {
			t.Fatal(err)
		}
		// log.I.F("len %d\n%s\n", len(b), eb.SerializeIndented())
		counter++
		out = out[:0]
		// break
	}
	chk.E(scanner.Err())
	t.Logf(
		"unmarshaled json, marshaled binary, unmarshaled binary, %d events in %v av %v per event",
		counter, time.Since(now), time.Since(now)/time.Duration(counter),
	)
}
