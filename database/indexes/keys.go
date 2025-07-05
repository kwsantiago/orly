package indexes

import (
	"io"
	"not.realy.lol/database/indexes/types"
	"reflect"

	"not.realy.lol/chk"
	"not.realy.lol/interfaces/codec"
)

var counter int

func init() {
	// Initialize counter to ensure it starts from 0
	counter = 0
}

func next() int { counter++; return counter - 1 }

type P struct {
	val []byte
}

func NewPrefix(prf ...int) (p *P) {
	if len(prf) > 0 {
		prefix := Prefix(prf[0])
		if prefix == "" {
			// If the prefix is empty, use a default prefix
			return &P{[]byte("def")}
		}
		return &P{[]byte(prefix)}
	} else {
		return &P{[]byte{0, 0, 0}}
	}
}

func (p *P) Bytes() (b []byte) { return p.val }

func (p *P) MarshalWrite(w io.Writer) (err error) {
	_, err = w.Write(p.val)
	return
}

func (p *P) UnmarshalRead(r io.Reader) (err error) {
	// Allocate a buffer for val if it's nil or empty
	if p.val == nil || len(p.val) == 0 {
		p.val = make([]byte, 3) // Prefixes are 3 bytes
	}
	_, err = r.Read(p.val)
	return
}

type I string

func (i I) Write(w io.Writer) (n int, err error) { return w.Write([]byte(i)) }

// Prefix returns the two byte human-readable prefixes that go in front of
// database indexes.
func Prefix(prf int) (i I) {
	switch prf {
	case Event:
		return "evt"
	case Id:
		return "eid"
	case IdPubkeyCreatedAt:
		return "ipc"
	case PubkeyCreatedAt:
		return "pca"
	case CreatedAt:
		return "ica"
	case PubkeyTagCreatedAt:
		return "ptc"
	case TagCreatedAt:
		return "itc"
	case Kind:
		return "iki"
	case KindCreatedAt:
		return "kca"
	case KindPubkey:
		return "kpk"
	case KindPubkeyCreatedAt:
		return "kpc"
	case KindTag:
		return "ikt"
	case KindTagCreatedAt:
		return "ktc"
	case KindPubkeyTagCreatedAt:
		return "kpt"
	}
	return
}

type Encs []codec.I

// T is a wrapper around an array of codec.I. The caller provides the Encs so
// they can then call the accessor function of the codec.I implementation.
type T struct{ Encs }

// New creates a new indexes. The helper functions below have an encode and
// decode variant, the decode variant does not add the prefix encoder because it
// has been read by Identify.
func New(encoders ...codec.I) (i *T) { return &T{encoders} }

func (t *T) MarshalWrite(w io.Writer) (err error) {
	for _, e := range t.Encs {
		if e == nil || reflect.ValueOf(e).IsNil() {
			// Skip nil encoders instead of returning early
			continue
		}
		if err = e.MarshalWrite(w); chk.E(err) {
			return
		}
	}
	return
}

func (t *T) UnmarshalRead(r io.Reader) (err error) {
	for _, e := range t.Encs {
		if err = e.UnmarshalRead(r); chk.E(err) {
			return
		}
	}
	return
}

// Event is the whole event stored in binary format
//
//	[ prefix ][ 8 byte serial ] [ event in binary format ]
var Event = next()

func EventVars() (ser *types.Uint40) {
	ser = new(types.Uint40)
	return
}
func EventEnc(ser *types.Uint40) (enc *T) {
	return New(NewPrefix(Event), ser)
}
func EventDec(ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), ser)
}

// Id contains a truncated 8-byte hash of an event index. This is the secondary
// key of an event, the primary key is the serial found in the Event.
//
// [ prefix ][ 8 bytes truncated hash of Id ][ 8 serial ]
var Id = next()

func IdVars() (id *types.IdHash, ser *types.Uint40) {
	id = new(types.IdHash)
	ser = new(types.Uint40)
	return
}
func IdEnc(id *types.IdHash, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(Id), id, ser)
}
func IdSearch(id *types.IdHash) (enc *T) {
	return New(NewPrefix(Id), id)
}
func IdDec(id *types.IdHash, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), id, ser)
}

// IdPubkeyCreatedAt is an index designed to enable sorting and filtering of
// results found via other indexes, without having to decode the event.
//
// [ prefix ][ 8 bytes serial ][ 32 bytes full event ID ][ 8 bytes truncated hash of pubkey ][ 8 bytes timestamp ]
var IdPubkeyCreatedAt = next()

func IdPubkeyCreatedAtVars() (
	ser *types.Uint40, t *types.Id, p *types.PubHash, ca *types.Uint64,
) {
	ser = new(types.Uint40)
	t = new(types.Id)
	p = new(types.PubHash)
	ca = new(types.Uint64)
	return
}
func IdPubkeyCreatedAtEnc(
	ser *types.Uint40, t *types.Id, p *types.PubHash, ca *types.Uint64,
) (enc *T) {
	return New(NewPrefix(IdPubkeyCreatedAt), ser, t, p, ca)
}
func IdPubkeyCreatedAtSearch(ser *types.Uint40) (enc *T) {
	return New(NewPrefix(IdPubkeyCreatedAt), ser)
}
func IdPubkeyCreatedAtDec(
	ser *types.Uint40, t *types.Id, p *types.PubHash, ca *types.Uint64,
) (enc *T) {
	return New(NewPrefix(), ser, t, p, ca)
}

// CreatedAt is an index that allows search for the timestamp on the event.
//
// [ prefix ][ timestamp 8 bytes timestamp ][ 8 serial ]
var CreatedAt = next()

func CreatedAtVars() (ca *types.Uint64, ser *types.Uint40) {
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func CreatedAtEnc(ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(CreatedAt), ca, ser)
}
func CreatedAtDec(ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), ca, ser)
}

// PubkeyCreatedAt is a composite index that allows search by pubkey
// filtered by timestamp.
//
// [ prefix ][ 8 bytes truncated hash of pubkey ][ 8 bytes timestamp ][ 8 serial ]
var PubkeyCreatedAt = next()

func PubkeyCreatedAtVars() (
	p *types.PubHash, ca *types.Uint64, ser *types.Uint40,
) {
	p = new(types.PubHash)
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func PubkeyCreatedAtEnc(
	p *types.PubHash, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(PubkeyCreatedAt), p, ca, ser)
}
func PubkeyCreatedAtDec(
	p *types.PubHash, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), p, ca, ser)
}

// PubkeyTagCreatedAt allows searching for a pubkey, tag and timestamp.
//
// [ prefix ][ 8 bytes truncated hash of pubkey ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 bytes timestamp ][ 8 serial ]
var PubkeyTagCreatedAt = next()

func PubkeyTagCreatedAtVars() (
	p *types.PubHash, k *types.Letter, v *types.Ident, ca *types.Uint64,
	ser *types.Uint40,
) {
	p = new(types.PubHash)
	k = new(types.Letter)
	v = new(types.Ident)
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func PubkeyTagCreatedAtEnc(
	p *types.PubHash, k *types.Letter, v *types.Ident, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(PubkeyTagCreatedAt), p, k, v, ca, ser)
}
func PubkeyTagCreatedAtDec(
	p *types.PubHash, k *types.Letter, v *types.Ident, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), p, k, v, ca, ser)
}

// TagCreatedAt allows searching for a tag and filter by timestamp.
//
// [ prefix ][ 8 bytes truncated hash of pubkey ][ 8 bytes truncated hash of value ][ 8 bytes timestamp ][ 8 serial ]
var TagCreatedAt = next()

func TagCreatedAtVars() (
	k *types.Letter, v *types.Ident, ca *types.Uint64, ser *types.Uint40,
) {
	k = new(types.Letter)
	v = new(types.Ident)
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func TagCreatedAtEnc(
	k *types.Letter, v *types.Ident, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(TagCreatedAt), k, v, ca, ser)
}
func TagCreatedAtDec(
	k *types.Letter, v *types.Ident, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), k, v, ca, ser)
}

// Kind
//
// [ prefix ][ 2 byte kind ][ 8 byte serial ]
var Kind = next()

func KindVars() (ki *types.Uint16, ser *types.Uint40) {
	ki = new(types.Uint16)
	ser = new(types.Uint40)
	return
}
func KindEnc(ki *types.Uint16, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(Kind), ki, ser)

}
func KindDec(ki *types.Uint16, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), ki, ser)
}

// KindPubkey
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of pubkey ][ 8 byte serial ]
var KindPubkey = next()

func KindPubkeyVars() (ki *types.Uint16, p *types.PubHash, ser *types.Uint40) {
	ki = new(types.Uint16)
	p = new(types.PubHash)
	ser = new(types.Uint40)
	return
}
func KindPubkeyEnc(
	ki *types.Uint16, p *types.PubHash, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkey), ki, p, ser)
}
func KindPubkeyDec(
	ki *types.Uint16, p *types.PubHash, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, ser)
}

// KindCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes timestamp ][ 8 byte serial ]
var KindCreatedAt = next()

func KindCreatedAtVars() (
	ki *types.Uint16, ca *types.Uint64, ser *types.Uint40,
) {
	ki = new(types.Uint16)
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func KindCreatedAtEnc(
	ki *types.Uint16, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(KindCreatedAt), ki, ca, ser)

}
func KindCreatedAtDec(
	ki *types.Uint16, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, ca, ser)

}

// KindTag
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 byte serial ]
var KindTag = next()

func KindTagVars() (
	ki *types.Uint16, k *types.Letter, v *types.Ident, ser *types.Uint40,
) {
	ki = new(types.Uint16)
	k = new(types.Letter)
	v = new(types.Ident)
	ser = new(types.Uint40)
	return
}
func KindTagEnc(
	ki *types.Uint16, k *types.Letter, v *types.Ident, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(KindTag), ki, k, v, ser)
}
func KindTagDec(
	ki *types.Uint16, k *types.Letter, v *types.Ident, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, k, v, ser)
}

// KindTagCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 byte serial ]
var KindTagCreatedAt = next()

func KindTagCreatedAtVars() (
	ki *types.Uint16, k *types.Letter, v *types.Ident, ca *types.Uint64,
	ser *types.Uint40,
) {
	ki = new(types.Uint16)
	k = new(types.Letter)
	v = new(types.Ident)
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func KindTagCreatedAtEnc(
	ki *types.Uint16, k *types.Letter, v *types.Ident, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(KindTagCreatedAt), ki, k, v, ca, ser)
}
func KindTagCreatedAtDec(
	ki *types.Uint16, k *types.Letter, v *types.Ident, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, k, v, ca, ser)
}

// KindPubkeyCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of pubkey ][ 8 bytes timestamp ][ 8 byte serial ]
var KindPubkeyCreatedAt = next()

func KindPubkeyCreatedAtVars() (
	ki *types.Uint16, p *types.PubHash, k *types.Letter, v *types.Ident,
	ca *types.Uint64,
	ser *types.Uint40,
) {
	ki = new(types.Uint16)
	k = new(types.Letter)
	v = new(types.Ident)
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func KindPubkeyCreatedAtEnc(
	ki *types.Uint16, p *types.PubHash, k *types.Letter, v *types.Ident,
	ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkeyCreatedAt), ki, p, k, v, ca, ser)
}
func KindPubkeyCreatedAtDec(
	ki *types.Uint16, p *types.PubHash, k *types.Letter, v *types.Ident,
	ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, k, v, ca, ser)
}

// KindPubkeyTagCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of pubkey ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 bytes timestamp ][ 8 byte serial ]
var KindPubkeyTagCreatedAt = next()

func KindPubkeyTagCreatedAtVars() (
	ki *types.Uint16, p *types.PubHash, k *types.Letter, v *types.Ident,
	ca *types.Uint64,
	ser *types.Uint40,
) {
	ki = new(types.Uint16)
	k = new(types.Letter)
	v = new(types.Ident)
	ca = new(types.Uint64)
	ser = new(types.Uint40)
	return
}
func KindPubkeyTagCreatedAtEnc(
	ki *types.Uint16, p *types.PubHash, k *types.Letter, v *types.Ident,
	ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkeyTagCreatedAt), ki, p, k, v, ca, ser)
}
func KindPubkeyTagCreatedAtDec(
	ki *types.Uint16, p *types.PubHash, k *types.Letter, v *types.Ident,
	ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, k, v, ca, ser)
}
