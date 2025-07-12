// Package tests provides a tool to generate arbitrary random events for fuzz
// testing the encoder.
package tests

import (
	"encoding/base64"
	"orly.dev/chk"

	"lukechampine.com/frand"

	"orly.dev/event"
	"orly.dev/kind"
	"orly.dev/p256k"
	"orly.dev/timestamp"
)

// GenerateEvent creates events full of random kinds and content data.
func GenerateEvent(maxSize int) (ev *event.E, binSize int, err error) {
	l := frand.Intn(maxSize * 6 / 8) // account for base64 expansion
	ev = &event.E{
		Kind:      kind.TextNote,
		CreatedAt: timestamp.Now(),
		Content:   []byte(base64.StdEncoding.EncodeToString(frand.Bytes(l))),
	}
	signer := new(p256k.Signer)
	if err = signer.Generate(); chk.E(err) {
		return
	}
	if err = ev.Sign(signer); chk.E(err) {
		return
	}
	var bin []byte
	bin = ev.Marshal(bin)
	binSize = len(bin)
	return
}
