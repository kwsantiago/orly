package indexes

import (
	"io"
	"reflect"

	"not.realy.lol/chk"
	"not.realy.lol/database/indexes/types/fullid"
	"not.realy.lol/database/indexes/types/identhash"
	"not.realy.lol/database/indexes/types/idhash"
	. "not.realy.lol/database/indexes/types/number"
	"not.realy.lol/database/indexes/types/pubhash"
	"not.realy.lol/interfaces/codec"
)

var counter int

func next() int { counter++; return counter - 1 }

type P struct {
	val []byte
}

func NewPrefix(prf ...int) (p *P) {
	if len(prf) > 0 {
		return &P{[]byte(Prefix(prf[0]))}
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
			// allow a field to be empty, as is needed for search indexes to
			// create search
			return
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

func EventVars() (ser *Uint40) {
	ser = new(Uint40)
	return
}
func EventEnc(ser *Uint40) (enc *T) {
	return New(NewPrefix(Event), ser)
}
func EventDec(ser *Uint40) (enc *T) {
	return New(NewPrefix(), ser)
}

// Id contains a truncated 8-byte hash of an event index. This is the secondary
// key of an event, the primary key is the serial found in the Event.
//
// [ prefix ][ 8 bytes truncated hash of Id ][ 8 serial ]
var Id = next()

func IdVars() (id *idhash.T, ser *Uint40) {
	id = idhash.New()
	ser = new(Uint40)
	return
}
func IdEnc(id *idhash.T, ser *Uint40) (enc *T) {
	return New(NewPrefix(Id), id, ser)
}
func IdSearch(id *idhash.T) (enc *T) {
	return New(NewPrefix(Id), id)
}
func IdDec(id *idhash.T, ser *Uint40) (enc *T) {
	return New(NewPrefix(), id, ser)
}

// IdPubkeyCreatedAt is an index designed to enable sorting and filtering of
// results found via other indexes, without having to decode the event.
//
// [ prefix ][ 8 bytes serial ][ 32 bytes full event ID ][ 8 bytes truncated hash of pubkey ][ 8 bytes timestamp ]
var IdPubkeyCreatedAt = next()

func IdPubkeyCreatedAtVars() (
	ser *Uint40, t *fullid.T, p *pubhash.T, ca *Uint64,
) {
	ser = new(Uint40)
	t = new(fullid.T)
	p = new(pubhash.T)
	ca = new(Uint64)
	return
}
func IdPubkeyCreatedAtEnc(
	ser *Uint40, t *fullid.T, p *pubhash.T, ca *Uint64,
) (enc *T) {
	return New(NewPrefix(IdPubkeyCreatedAt), ser, t, p, ca)
}
func IdPubkeyCreatedAtSearch(ser *Uint40) (enc *T) {
	return New(NewPrefix(IdPubkeyCreatedAt), ser)
}
func IdPubkeyCreatedAtDec(
	ser *Uint40, t *fullid.T, p *pubhash.T, ca *Uint64,
) (enc *T) {
	return New(NewPrefix(), ser, t, p, ca)
}

// CreatedAt is an index that allows search for the timestamp on the event.
//
// [ prefix ][ timestamp 8 bytes timestamp ][ 8 serial ]
var CreatedAt = next()

func CreatedAtVars() (ca *Uint64, ser *Uint40) {
	ca = new(Uint64)
	ser = new(Uint40)
	return
}
func CreatedAtEnc(ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(CreatedAt), ca, ser)
}
func CreatedAtDec(ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(), ca, ser)
}

// PubkeyCreatedAt is a composite index that allows search by pubkey
// filtered by timestamp.
//
// [ prefix ][ 8 bytes truncated hash of pubkey ][ 8 bytes timestamp ][ 8 serial ]
var PubkeyCreatedAt = next()

func PubkeyCreatedAtVars() (p *pubhash.T, ca *Uint64, ser *Uint40) {
	p = new(pubhash.T)
	ca = new(Uint64)
	ser = new(Uint40)
	return
}
func PubkeyCreatedAtEnc(p *pubhash.T, ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(PubkeyCreatedAt), p, ca, ser)
}
func PubkeyCreatedAtDec(p *pubhash.T, ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(), p, ca, ser)
}

// PubkeyTagCreatedAt allows searching for a pubkey, tag and timestamp.
//
// [ prefix ][ 8 bytes truncated hash of pubkey ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 bytes timestamp ][ 8 serial ]
var PubkeyTagCreatedAt = next()

func PubkeyTagCreatedAtVars() (
	p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) {
	p = new(pubhash.T)
	k = new(identhash.T)
	v = new(identhash.T)
	ca = new(Uint64)
	ser = new(Uint40)
	return
}
func PubkeyTagCreatedAtEnc(
	p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(PubkeyTagCreatedAt), p, k, v, ca, ser)
}
func PubkeyTagCreatedAtDec(
	p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(), p, k, v, ca, ser)
}

// TagCreatedAt allows searching for a tag and filter by timestamp.
//
// [ prefix ][ 8 bytes truncated hash of pubkey ][ 8 bytes truncated hash of value ][ 8 bytes timestamp ][ 8 serial ]
var TagCreatedAt = next()

func TagCreatedAtVars() (k, v *identhash.T, ca *Uint64, ser *Uint40) {
	k = new(identhash.T)
	v = new(identhash.T)
	ca = new(Uint64)
	ser = new(Uint40)
	return
}
func TagCreatedAtEnc(k, v *identhash.T, ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(TagCreatedAt), k, v, ca, ser)
}
func TagCreatedAtDec(k, v *identhash.T, ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(), k, v, ca, ser)
}

// Kind
//
// [ prefix ][ 2 byte kind ][ 8 byte serial ]
var Kind = next()

func KindVars() (ki *Uint16, ser *Uint40) {
	ki = new(Uint16)
	ser = new(Uint40)
	return
}
func KindEnc(ki *Uint16, ser *Uint40) (enc *T) {
	return New(NewPrefix(Kind), ki, ser)

}
func KindDec(ki *Uint16, ser *Uint40) (enc *T) {
	return New(NewPrefix(), ki, ser)
}

// KindPubkey
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of pubkey ][ 8 byte serial ]
var KindPubkey = next()

func KindPubkeyVars() (ki *Uint16, p *pubhash.T, ser *Uint40) {
	ki = new(Uint16)
	p = new(pubhash.T)
	ser = new(Uint40)
	return
}
func KindPubkeyEnc(ki *Uint16, p *pubhash.T, ser *Uint40) (enc *T) {
	return New(NewPrefix(KindPubkey), ki, p, ser)
}
func KindPubkeyDec(ki *Uint16, p *pubhash.T, ser *Uint40) (enc *T) {
	return New(NewPrefix(), ki, p, ser)
}

// KindCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes timestamp ][ 8 byte serial ]
var KindCreatedAt = next()

func KindCreatedAtVars() (ki *Uint16, ca *Uint64, ser *Uint40) {
	ki = new(Uint16)
	ca = new(Uint64)
	ser = new(Uint40)
	return
}
func KindCreatedAtEnc(ki *Uint16, ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(KindCreatedAt), ki, ca, ser)

}
func KindCreatedAtDec(ki *Uint16, ca *Uint64, ser *Uint40) (enc *T) {
	return New(NewPrefix(), ki, ca, ser)

}

// KindTag
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 byte serial ]
var KindTag = next()

func KindTagVars() (ki *Uint16, k, v *identhash.T, ser *Uint40) {
	ki = new(Uint16)
	k = new(identhash.T)
	v = new(identhash.T)
	ser = new(Uint40)
	return
}
func KindTagEnc(ki *Uint16, k, v *identhash.T, ser *Uint40) (enc *T) {
	return New(NewPrefix(KindTag), ki, k, v, ser)
}
func KindTagDec(ki *Uint16, k, v *identhash.T, ser *Uint40) (enc *T) {
	return New(NewPrefix(), ki, k, v, ser)
}

// KindTagCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 byte serial ]
var KindTagCreatedAt = next()

func KindTagCreatedAtVars() (
	ki *Uint16, k, v *identhash.T, ca *Uint64, ser *Uint40,
) {
	ki = new(Uint16)
	k = new(identhash.T)
	v = new(identhash.T)
	ser = new(Uint40)
	return
}
func KindTagCreatedAtEnc(
	ki *Uint16, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(KindTagCreatedAt), ki, k, v, ca, ser)
}
func KindTagCreatedAtDec(
	ki *Uint16, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, k, v, ca, ser)
}

// KindPubkeyCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of pubkey ][ 8 bytes timestamp ][ 8 byte serial ]
var KindPubkeyCreatedAt = next()

func KindPubkeyCreatedAtVars() (
	ki *Uint16, p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) {
	ki = new(Uint16)
	k = new(identhash.T)
	v = new(identhash.T)
	ca = new(Uint64)
	ser = new(Uint40)
	return
}
func KindPubkeyCreatedAtEnc(
	ki *Uint16, p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkeyCreatedAt), ki, p, k, v, ser)

}
func KindPubkeyCreatedAtDec(
	ki *Uint16, p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, k, v, ser)
}

// KindPubkeyTagCreatedAt
//
// [ prefix ][ 2 byte kind ][ 8 bytes truncated hash of pubkey ][ 8 bytes truncated hash of key ][ 8 bytes truncated hash of value ][ 8 bytes timestamp ][ 8 byte serial ]
var KindPubkeyTagCreatedAt = next()

func KindPubkeyTagCreatedAtVars() (
	ki *Uint16, p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) {
	ki = new(Uint16)
	k = new(identhash.T)
	v = new(identhash.T)
	ca = new(Uint64)
	ser = new(Uint40)
	return
}
func KindPubkeyTagCreatedAtEnc(
	ki *Uint16, p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkeyTagCreatedAt), ki, p, k, v, ca, ser)
}
func KindPubkeyTagCreatedAtDec(
	ki *Uint16, p *pubhash.T, k, v *identhash.T, ca *Uint64, ser *Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, k, v, ca, ser)
}
