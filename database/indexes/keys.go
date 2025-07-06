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
	EventPrefix         = I("evt")
	IdPrefix            = I("eid")
	IdPubkeyPrefix      = I("ipc")
	PubkeyPrefix        = I("pca")
	CreatedAtPrefix     = I("ica")
	PubkeyTagPrefix     = I("ptc")
	TagPrefix           = I("itc")
	KindPrefix          = I("kca")
	KindPubkeyPrefix    = I("kpc")
	KindTagPrefix       = I("ktc")
	KindPubkeyTagPrefix = I("kpt")
)

// Prefix returns the three byte human-readable prefixes that go in front of
// database indexes.
func Prefix(prf int) (i I) {
	switch prf {
	case Event:
		return EventPrefix
	case Id:
		return IdPrefix
	case IdPubkey:
		return IdPubkeyPrefix
	case Pubkey:
		return PubkeyPrefix
	case CreatedAt:
		return CreatedAtPrefix
	case PubkeyTag:
		return PubkeyTagPrefix
	case Tag:
		return TagPrefix
	case Kind:
		return KindPrefix
	case KindPubkey:
		return KindPubkeyPrefix
	case KindTag:
		return KindTagPrefix
	case KindPubkeyTag:
		return KindPubkeyTagPrefix
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
	case IdPubkeyPrefix:
		i = IdPubkey
	case PubkeyPrefix:
		i = Pubkey
	case CreatedAtPrefix:
		i = CreatedAt
	case PubkeyTagPrefix:
		i = PubkeyTag
	case TagPrefix:
		i = Tag
	case KindPrefix:
		i = Kind
	case KindPubkeyPrefix:
		i = KindPubkey
	case KindTagPrefix:
		i = KindTag
	case KindPubkeyTagPrefix:
		i = KindPubkeyTag
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

// IdPubkey is an index designed to enable sorting and filtering of
// results found via other indexes, without having to decode the event.
//
//	3 prefix|5 serial|32 Id|8 pubkey hash|8 timestamp
var IdPubkey = next()

func IdPubkeyVars() (
	ser *t.Uint40, fid *t.Id, p *t.PubHash, ca *t.Uint64,
) {
	return new(t.Uint40), new(t.Id), new(t.PubHash), new(t.Uint64)
}
func IdPubkeyEnc(
	ser *t.Uint40, fid *t.Id, p *t.PubHash, ca *t.Uint64,
) (enc *T) {
	return New(NewPrefix(IdPubkey), ser, fid, p, ca)
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

// PubkeyTag allows searching for a pubkey, tag and timestamp.
//
//	3 prefix|8 pubkey hash|1 key letter|8 value hash|8 timestamp|5 serial
var PubkeyTag = next()

func PubkeyTagVars() (
	p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) {
	return new(t.PubHash), new(t.Letter), new(t.Ident), new(t.Uint64),
		new(t.Uint40)
}
func PubkeyTagEnc(
	p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(PubkeyTag), p, k, v, ca, ser)
}
func PubkeyTagDec(
	p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), p, k, v, ca, ser)
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

// KindTag
//
//	3 prefix|2 kind|1 key letter|8 value hash|8 timestamp|5 serial
var KindTag = next()

func KindTagVars() (
	ki *t.Uint16, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) {
	return new(t.Uint16), new(t.Letter), new(t.Ident), new(t.Uint64),
		new(t.Uint40)
}
func KindTagEnc(
	ki *t.Uint16, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(KindTag), ki, k, v, ca, ser)
}
func KindTagDec(
	ki *t.Uint16, k *t.Letter, v *t.Ident, ca *t.Uint64, ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, k, v, ca, ser)
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

// KindPubkeyTag
//
//	3 prefix|2 kind|8 pubkey hash|1 key letter|8 value hash|8 bytes timestamp|5 byte serial
var KindPubkeyTag = next()

func KindPubkeyTagVars() (
	ki *t.Uint16, p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64,
	ser *t.Uint40,
) {
	return new(t.Uint16), new(t.PubHash), new(t.Letter), new(t.Ident),
		new(t.Uint64), new(t.Uint40)
}
func KindPubkeyTagEnc(
	ki *t.Uint16, p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64,
	ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkeyTag), ki, p, k, v, ca, ser)
}
func KindPubkeyTagDec(
	ki *t.Uint16, p *t.PubHash, k *t.Letter, v *t.Ident, ca *t.Uint64,
	ser *t.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, k, v, ca, ser)
}
