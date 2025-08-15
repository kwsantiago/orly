package event

import (
	"bytes"
	"fmt"
	"io"
	"orly.dev/pkg/crypto/ec/schnorr"
	"orly.dev/pkg/crypto/sha256"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tags"
	text2 "orly.dev/pkg/encoders/text"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/errorf"
)

func (ev *E) UnmarshalRead(rd io.Reader) (err error) {
	key := make([]byte, 0, 9)
	// Read entire content from the io.Reader into a buffer to reuse existing slice-based parser.
	var b []byte
	if rd != nil {
		var readErr error
		b, readErr = io.ReadAll(rd)
		if readErr != nil {
			return readErr
		}
	}
	r := b
	for ; len(r) > 0; r = r[1:] {
		// Skip whitespace
		if isWhitespace(r[0]) {
			continue
		}
		if r[0] == '{' {
			r = r[1:]
			goto BetweenKeys
		}
	}
	goto eof
BetweenKeys:
	for ; len(r) > 0; r = r[1:] {
		// Skip whitespace
		if isWhitespace(r[0]) {
			continue
		}
		if r[0] == '"' {
			r = r[1:]
			goto InKey
		}
	}
	goto eof
InKey:
	for ; len(r) > 0; r = r[1:] {
		if r[0] == '"' {
			r = r[1:]
			goto InKV
		}
		key = append(key, r[0])
	}
	goto eof
InKV:
	for ; len(r) > 0; r = r[1:] {
		// Skip whitespace
		if isWhitespace(r[0]) {
			continue
		}
		if r[0] == ':' {
			r = r[1:]
			goto InVal
		}
	}
	goto eof
InVal:
	// Skip whitespace before value
	for len(r) > 0 && isWhitespace(r[0]) {
		r = r[1:]
	}

	switch key[0] {
	case jId[0]:
		if !bytes.Equal(jId, key) {
			goto invalid
		}
		var id []byte
		if id, r, err = text2.UnmarshalHex(r); chk.E(err) {
			return
		}
		if len(id) != sha256.Size {
			err = errorf.E(
				"invalid ID, require %d got %d", sha256.Size,
				len(id),
			)
			return
		}
		ev.ID = id
		goto BetweenKV
	case jPubkey[0]:
		if !bytes.Equal(jPubkey, key) {
			goto invalid
		}
		var pk []byte
		if pk, r, err = text2.UnmarshalHex(r); chk.E(err) {
			return
		}
		if len(pk) != schnorr.PubKeyBytesLen {
			err = errorf.E(
				"invalid pubkey, require %d got %d",
				schnorr.PubKeyBytesLen, len(pk),
			)
			return
		}
		ev.Pubkey = pk
		goto BetweenKV
	case jKind[0]:
		if !bytes.Equal(jKind, key) {
			goto invalid
		}
		ev.Kind = kind.New(0)
		if r, err = ev.Kind.Unmarshal(r); chk.E(err) {
			return
		}
		goto BetweenKV
	case jTags[0]:
		if !bytes.Equal(jTags, key) {
			goto invalid
		}
		ev.Tags = tags.New()
		if r, err = ev.Tags.Unmarshal(r); chk.E(err) {
			return
		}
		goto BetweenKV
	case jSig[0]:
		if !bytes.Equal(jSig, key) {
			goto invalid
		}
		var sig []byte
		if sig, r, err = text2.UnmarshalHex(r); chk.E(err) {
			return
		}
		if len(sig) != schnorr.SignatureSize {
			err = errorf.E(
				"invalid sig length, require %d got %d '%s'\n%s",
				schnorr.SignatureSize, len(sig), r, b,
			)
			return
		}
		ev.Sig = sig
		goto BetweenKV
	case jContent[0]:
		if key[1] == jContent[1] {
			if !bytes.Equal(jContent, key) {
				goto invalid
			}
			if ev.Content, r, err = text2.UnmarshalQuoted(r); chk.T(err) {
				return
			}
			goto BetweenKV
		} else if key[1] == jCreatedAt[1] {
			if !bytes.Equal(jCreatedAt, key) {
				goto invalid
			}
			ev.CreatedAt = timestamp.New(int64(0))
			if r, err = ev.CreatedAt.Unmarshal(r); chk.T(err) {
				return
			}
			goto BetweenKV
		} else {
			goto invalid
		}
	default:
		goto invalid
	}
BetweenKV:
	key = key[:0]
	for ; len(r) > 0; r = r[1:] {
		// Skip whitespace
		if isWhitespace(r[0]) {
			continue
		}

		switch {
		case len(r) == 0:
			return
		case r[0] == '}':
			r = r[1:]
			goto AfterClose
		case r[0] == ',':
			r = r[1:]
			goto BetweenKeys
		case r[0] == '"':
			r = r[1:]
			goto InKey
		}
	}
	goto eof
AfterClose:
	// Skip any trailing whitespace
	for len(r) > 0 && isWhitespace(r[0]) {
		r = r[1:]
	}
	return
invalid:
	err = fmt.Errorf(
		"invalid key,\n'%s'\n'%s'\n'%s'", string(b), string(b[:len(r)]),
		string(r),
	)
	return
eof:
	err = io.EOF
	return
}
