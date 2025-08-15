package event

import (
	"io"
	"orly.dev/pkg/encoders/hex"
	text2 "orly.dev/pkg/encoders/text"
)

// MarshalWrite writes the JSON representation of an event to an io.Writer. This
// is a version of MarshalWithWhitespace that writes to an io.Writer instead of
// appending to a slice, with whitespace disabled by default. It implements the
// codec.I interface.
func (ev *E) MarshalWrite(w io.Writer) (err error) {
	return ev.MarshalWriteWithWhitespace(w, false)
}

// MarshalWriteWithWhitespace writes the JSON representation of an event to an
// io.Writer with optional whitespace formatting. If the 'on' flag is set to
// true, it adds tabs and newlines to make the JSON more readable for humans.
// This is a version of MarshalWithWhitespace that writes to an io.Writer
// instead of appending to a slice.
func (ev *E) MarshalWriteWithWhitespace(w io.Writer, on bool) (err error) {
	// open parentheses
	if _, err = w.Write([]byte{'{'}); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n', '\t'}); err != nil {
			return
		}
	}
	var buf []byte
	buf = text2.JSONKey(buf, jId)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{' '}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.AppendQuote(buf, ev.ID, hex.EncAppend)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if _, err = w.Write([]byte{','}); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n', '\t'}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.JSONKey(buf, jPubkey)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{' '}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.AppendQuote(buf, ev.Pubkey, hex.EncAppend)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if _, err = w.Write([]byte{','}); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n', '\t'}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.JSONKey(buf, jCreatedAt)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{' '}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = ev.CreatedAt.Marshal(buf)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if _, err = w.Write([]byte{','}); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n', '\t'}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.JSONKey(buf, jKind)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{' '}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = ev.Kind.Marshal(buf)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if _, err = w.Write([]byte{','}); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n', '\t'}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.JSONKey(buf, jTags)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{' '}); err != nil {
			return
		}
	}
	buf = buf[:0]
	if on {
		buf = ev.Tags.MarshalWithWhitespace(buf)
	} else {
		buf = ev.Tags.Marshal(buf)
	}
	if _, err = w.Write(buf); err != nil {
		return
	}
	if _, err = w.Write([]byte{','}); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n', '\t'}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.JSONKey(buf, jContent)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{' '}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.AppendQuote(buf, ev.Content, text2.NostrEscape)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if _, err = w.Write([]byte{','}); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n', '\t'}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.JSONKey(buf, jSig)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{' '}); err != nil {
			return
		}
	}
	buf = buf[:0]
	buf = text2.AppendQuote(buf, ev.Sig, hex.EncAppend)
	if _, err = w.Write(buf); err != nil {
		return
	}
	if on {
		if _, err = w.Write([]byte{'\n'}); err != nil {
			return
		}
	}
	if _, err = w.Write([]byte{'}'}); err != nil {
		return
	}
	return
}
