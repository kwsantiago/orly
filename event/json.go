package event

import (
	"bytes"
	"github.com/minio/sha256-simd"
	"io"
	"orly.dev/chk"
	"orly.dev/ec/schnorr"
	"orly.dev/errorf"
	"orly.dev/hex"
	"orly.dev/kind"
	"orly.dev/tags"
	"orly.dev/text"
	"orly.dev/timestamp"
)

var (
	jId        = []byte("id")
	jPubkey    = []byte("pubkey")
	jCreatedAt = []byte("created_at")
	jKind      = []byte("kind")
	jTags      = []byte("tags")
	jContent   = []byte("content")
	jSig       = []byte("sig")
)

// Marshal appends an event.E to a provided destination slice.
func (ev *E) Marshal(dst []byte) (b []byte) {
	b = ev.MarshalWithWhitespace(dst, false)
	return
}

// MarshalWithWhitespace adds tabs and newlines to make the JSON more readable
// for humans, if the on flag is set to true.
func (ev *E) MarshalWithWhitespace(dst []byte, on bool) (b []byte) {
	// open parentheses
	dst = append(dst, '{')
	// Id
	if on {
		dst = append(dst, '\n', '\t')
	}
	dst = text.JSONKey(dst, jId)
	if on {
		dst = append(dst, ' ')
	}
	dst = text.AppendQuote(dst, ev.Id, hex.EncAppend)
	dst = append(dst, ',')
	// Pubkey
	if on {
		dst = append(dst, '\n', '\t')
	}
	dst = text.JSONKey(dst, jPubkey)
	if on {
		dst = append(dst, ' ')
	}
	dst = text.AppendQuote(dst, ev.Pubkey, hex.EncAppend)
	dst = append(dst, ',')
	if on {
		dst = append(dst, '\n', '\t')
	}
	// CreatedAt
	dst = text.JSONKey(dst, jCreatedAt)
	if on {
		dst = append(dst, ' ')
	}
	dst = ev.CreatedAt.Marshal(dst)
	dst = append(dst, ',')
	if on {
		dst = append(dst, '\n', '\t')
	}
	// Kind
	dst = text.JSONKey(dst, jKind)
	if on {
		dst = append(dst, ' ')
	}
	dst = ev.Kind.Marshal(dst)
	dst = append(dst, ',')
	if on {
		dst = append(dst, '\n', '\t')
	}
	// Tags
	dst = text.JSONKey(dst, jTags)
	if on {
		dst = append(dst, ' ')
	}
	if on {
		dst = ev.Tags.MarshalWithWhitespace(dst)
	} else {
		dst = ev.Tags.Marshal(dst)
	}
	dst = append(dst, ',')
	if on {
		dst = append(dst, '\n', '\t')
	}
	// Content
	dst = text.JSONKey(dst, jContent)
	if on {
		dst = append(dst, ' ')
	}
	dst = text.AppendQuote(dst, ev.Content, text.NostrEscape)
	dst = append(dst, ',')
	if on {
		dst = append(dst, '\n', '\t')
	}
	// jSig
	dst = text.JSONKey(dst, jSig)
	if on {
		dst = append(dst, ' ')
	}
	dst = text.AppendQuote(dst, ev.Sig, hex.EncAppend)
	if on {
		dst = append(dst, '\n')
	}
	// close parentheses
	dst = append(dst, '}')
	b = dst
	return
}

// Marshal is a normal function that is the same as event.E Marshal method
// except you explicitly specify the receiver.
func Marshal(ev *E, dst []byte) (b []byte) { return ev.Marshal(dst) }

// Unmarshal an event from JSON into an event.E.
// This function handles both minified and whitespace-formatted JSON.
func (ev *E) Unmarshal(b []byte) (r []byte, err error) {
	key := make([]byte, 0, 9)
	r = b
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
		if id, r, err = text.UnmarshalHex(r); chk.E(err) {
			return
		}
		if len(id) != sha256.Size {
			err = errorf.E(
				"invalid Id, require %d got %d", sha256.Size,
				len(id),
			)
			return
		}
		ev.Id = id
		goto BetweenKV
	case jPubkey[0]:
		if !bytes.Equal(jPubkey, key) {
			goto invalid
		}
		var pk []byte
		if pk, r, err = text.UnmarshalHex(r); chk.E(err) {
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
		if sig, r, err = text.UnmarshalHex(r); chk.E(err) {
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
			if ev.Content, r, err = text.UnmarshalQuoted(r); chk.T(err) {
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
	err = errorf.E(
		"invalid key,\n'%s'\n'%s'\n'%s'", string(b), string(b[:len(r)]),
		string(r),
	)
	return
eof:
	err = io.EOF
	return
}

// isWhitespace returns true if the byte is a whitespace character (space, tab, newline, carriage return).
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// Unmarshal is the same as the event.E Unmarshal method except you give it the
// event to marshal into instead of call it as a method of the type.
func Unmarshal(ev *E, b []byte) (r []byte, err error) { return ev.Unmarshal(b) }
