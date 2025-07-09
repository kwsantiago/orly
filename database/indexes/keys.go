package indexes

import (
	"io"
	t "orly.dev/database/indexes/types"
	"reflect"

	"orly.dev/chk"
	"orly.dev/interfaces/codec"
)

var counter int

func init() {
	// Initialize the counter to ensure it starts from 0
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
			panic("unknown prefix")
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

const (
	EventPrefix        = I("evt")
	IdPrefix           = I("eid")
	FullIdPubkeyPrefix = I("fpc")

	CreatedAtPrefix  = I("c--")
	KindPrefix       = I("kc-")
	PubkeyPrefix     = I("pc-")
	KindPubkeyPrefix = I("kpc")

	TagPrefix           = I("tc-")
	TagKindPrefix       = I("tkc")
	TagPubkeyPrefix     = I("tpc")
	TagKindPubkeyPrefix = I("tkp")
)

// Prefix returns the three byte human-readable prefixes that go in front of
// database indexes.
func Prefix(prf int) (i I) {
	switch prf {
	case Event:
		return EventPrefix
	case Id:
		return IdPrefix
	case FullIdPubkey:
		return FullIdPubkeyPrefix

	case CreatedAt:
		return CreatedAtPrefix
	case Kind:
		return KindPrefix
	case Pubkey:
		return PubkeyPrefix
	case KindPubkey:
		return KindPubkeyPrefix

	case Tag:
		return TagPrefix
	case TagKind:
		return TagKindPrefix
	case TagPubkey:
		return TagPubkeyPrefix
	case TagKindPubkey:
		return TagKindPubkeyPrefix
	}
	return
}

func Identify(r io.Reader) (i int, err error) {
	// this is here for completeness; however, searches don't need to identify
	// this as they work via generated prefixes made using Prefix.
	var b [3]byte
	_, err = r.Read(b[:])
	if err != nil {
		i = -1
		return
	}
	switch I(b[:]) {
	case EventPrefix:
		i = Event
	case IdPrefix:
		i = Id
	case FullIdPubkeyPrefix:
		i = FullIdPubkey

	case CreatedAtPrefix:
		i = CreatedAt
	case KindPrefix:
		i = Kind
	case PubkeyPrefix:
		i = Pubkey
	case KindPubkeyPrefix:
		i = KindPubkey

	case TagPrefix:
		i = Tag
	case TagKindPrefix:
		i = TagKind
	case TagPubkeyPrefix:
		i = TagPubkey
	case TagKindPubkeyPrefix:
		i = TagKindPubkey
	}
	return
}

type Encs []codec.I

// T is a wrapper around an array of codec.I. The caller provides the Encs so
// they can then call the accessor methods of the codec.I implementation.
type T struct{ Encs }

// New creates a new indexes.T. The helper functions below have an encode and
// decode variant, the decode variant does not add the prefix encoder because it
// has been read by Identify or just is being read, and found because it was
// written for the prefix in the iteration.
func New(encoders ...codec.I) (i *T) { return &T{encoders} }
func (t *T) MarshalWrite(w io.Writer) (err error) {
	for _, e := range t.Encs {
		if e == nil || reflect.ValueOf(e).IsNil() {
			// Skip nil encoders instead of returning early. This enables
			// generating search prefixes.
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
//	prefix|5 serial - event in binary format
var Event = next()

func EventVars() (ser *t.Uint40) { return new(t.Uint40) }
func EventEnc(ser *t.Uint40) (enc *T) {
	return New(NewPrefix(Event), ser)
}
func EventDec(ser *t.Uint40) (enc *T) { return New(NewPrefix(), ser) }

// Id contains a truncated 8-byte hash of an event index. This is the secondary
// key of an event, the primary key is the serial found in the Event.
//
//	3 prefix|8 Id hash|5 serial
var Id = next()

func IdVars() (id *t.IdHash, ser *t.Uint40) {
	return new(t.IdHash), new(t.Uint40)
}
func IdEnc(id *t.IdHash, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(Id), id, ser)
}
func IdDec(id *t.IdHash, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(), id, ser)
}

// FullIdPubkey is an index designed to enable sorting and filtering of
// results found via other indexes, without having to decode the event.
//
//	3 prefix|5 serial|32 Id|8 pubkey hash|8 timestamp
var FullIdPubkey = next()

func IdPubkeyVars() (
	ser *t.Uint40, fid *t.Id, p *t.PubHash, ca *t.Uint64,
) {
	return new(t.Uint40), new(t.Id), new(t.PubHash), new(t.Uint64)
}
func IdPubkeyEnc(
	ser *t.Uint40, fid *t.Id, p *t.PubHash, ca *t.Uint64,
) (enc *T) {
	return New(NewPrefix(FullIdPubkey), ser, fid, p, ca)
}
func IdPubkeyDec(
	ser *t.Uint40, fid *t.Id, p *t.PubHash, ca *t.Uint64,
) (enc *T) {
	return New(NewPrefix(), ser, fid, p, ca)
}

// CreatedAt is an index that allows search for the timestamp on the event.
//
//	3 prefix|8 timestamp|5 serial
var CreatedAt = next()

func CreatedAtVars() (ca *t.Uint64, ser *t.Uint40) {
	return new(t.Uint64), new(t.Uint40)
}
func CreatedAtEnc(ca *t.Uint64, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(CreatedAt), ca, ser)
}
func CreatedAtDec(ca *t.Uint64, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(), ca, ser)
}

// Kind
//
//	3 prefix|2 kind|8 timestamp|5 serial
var Kind = next()

func KindVars() (ki *t.Uint16, ca *t.Uint64, ser *t.Uint40) {
	return new(t.Uint16), new(t.Uint64), new(t.Uint40)
}
func KindEnc(ki *t.Uint16, ca *t.Uint64, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(Kind), ki, ca, ser)
}
func KindDec(ki *t.Uint16, ca *t.Uint64, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(), ki, ca, ser)
}

// Pubkey is a composite index that allows search by pubkey
// filtered by timestamp.
//
//	3 prefix|8 pubkey hash|8 timestamp|5 serial
var Pubkey = next()

func PubkeyVars() (p *t.PubHash, ca *t.Uint64, ser *t.Uint40) {
	return new(t.PubHash), new(t.Uint64), new(t.Uint40)
}
func PubkeyEnc(p *t.PubHash, ca *t.Uint64, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(Pubkey), p, ca, ser)
}
func PubkeyDec(p *t.PubHash, ca *t.Uint64, ser *t.Uint40) (enc *T) {
	return New(NewPrefix(), p, ca, ser)
}

// KindPubkey
//
//	3 prefix|2 kind|8 pubkey hash|8 timestamp|5 serial
var KindPubkey = next()

func KindPubkeyVars() (
	ki *t.Uint16, p *t.PubHash, ca *t.Uint64, ser *t.Uint40,
) {
	return new(t.Uint16), new(t.PubHash), new(t.Uint64), new(t.Uint40)
}
func KindPubkeyEnc(
	ki *t.Uint16, p *t.PubHash, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkey), ki, p, ca, ser)
}
func KindPubkeyDec(
	ki *t.Uint16, p *t.PubHash, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, ca, ser)
}

// Tag allows searching for a tag and filter by timestamp.
//
//	3 prefix|1 key letter|8 value hash|8 timestamp|5 serial
var Tag = next()

func TagVars() (
	k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) {
	return new(t.Letter), new(t.Ident), new(t.Uint64), new(t.Uint40)
}
func TagEnc(
	k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(Tag), k, v, ca, ser)
}
func TagDec(
	k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), k, v, ca, ser)
}

// TagKind
//
//	3 prefix|2 kind|1 key letter|8 value hash|8 timestamp|5 serial
var TagKind = next()

func TagKindVars() (
	ki *t.Uint16, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) {
	return new(t.Uint16), new(t.Letter), new(t.Ident), new(t.Uint64),
		new(t.Uint40)
}
func TagKindEnc(
	ki *t.Uint16, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(TagKind), ki, k, v, ca, ser)
}
func TagKindDec(
	ki *t.Uint16, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, k, v, ca, ser)
}

// TagPubkey allows searching for a pubkey, tag and timestamp.
//
//	3 prefix|8 pubkey hash|1 key letter|8 value hash|8 timestamp|5 serial
var TagPubkey = next()

func TagPubkeyVars() (
	p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) {
	return new(t.PubHash), new(t.Letter), new(t.Ident), new(t.Uint64),
		new(t.Uint40)
}
func TagPubkeyEnc(
	p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(TagPubkey), p, k, v, ca, ser)
}
func TagPubkeyDec(
	p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), p, k, v, ca, ser)
}

// TagKindPubkey
//
//	3 prefix|2 kind|8 pubkey hash|1 key letter|8 value hash|8 bytes timestamp|5 byte serial
var TagKindPubkey = next()

func TagKindPubkeyVars() (
	ki *t.Uint16, p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64,
	ser *t.Uint40,
) {
	return new(t.Uint16), new(t.PubHash), new(t.Letter), new(t.Ident),
		new(t.Uint64), new(t.Uint40)
}
func TagKindPubkeyEnc(
	ki *t.Uint16, p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64,
	ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(TagKindPubkey), ki, p, k, v, ca, ser)
}
func TagKindPubkeyDec(
	ki *t.Uint16, p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64,
	ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, k, v, ca, ser)
}
