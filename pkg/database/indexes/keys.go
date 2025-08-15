package indexes

import (
	"io"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/interfaces/codec"
	"orly.dev/pkg/utils/chk"
	"reflect"
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
	FullIdPubkeyPrefix = I("fpc") // full id, pubkey, created at

	CreatedAtPrefix  = I("c--") // created at
	KindPrefix       = I("kc-") // kind, created at
	PubkeyPrefix     = I("pc-") // pubkey, created at
	KindPubkeyPrefix = I("kpc") // kind, pubkey, created at

	TagPrefix           = I("tc-") // tag, created at
	TagKindPrefix       = I("tkc") // tag, kind, created at
	TagPubkeyPrefix     = I("tpc") // tag, pubkey, created at
	TagKindPubkeyPrefix = I("tkp") // tag, kind, pubkey, created at

	ExpirationPrefix = I("exp") // timestamp of expiration
	VersionPrefix    = I("ver") // database version number, for triggering reindexes when new keys are added (policy is add-only).
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

	case Expiration:
		return ExpirationPrefix
	case Version:
		return VersionPrefix
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

	case ExpirationPrefix:
		i = Expiration
	}
	return
}

type Encs []codec.I

// T is a wrapper around an array of codec.I. The caller provides the Encs so
// they can then call the accessor methods of the codec.I implementation.
type T struct{ Encs }

// New creates a new indexes.T. The helper functions below have an encode and
// decode variant, the decode variant doesn't add the prefix encoder because it
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

func EventVars() (ser *types.Uint40) { return new(types.Uint40) }
func EventEnc(ser *types.Uint40) (enc *T) {
	return New(NewPrefix(Event), ser)
}
func EventDec(ser *types.Uint40) (enc *T) { return New(NewPrefix(), ser) }

// Id contains a truncated 8-byte hash of an event index. This is the secondary
// key of an event, the primary key is the serial found in the Event.
//
//	3 prefix|8 ID hash|5 serial
var Id = next()

func IdVars() (id *types.IdHash, ser *types.Uint40) {
	return new(types.IdHash), new(types.Uint40)
}
func IdEnc(id *types.IdHash, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(Id), id, ser)
}
func IdDec(id *types.IdHash, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), id, ser)
}

// FullIdPubkey is an index designed to enable sorting and filtering of
// results found via other indexes, without having to decode the event.
//
//	3 prefix|5 serial|32 ID|8 pubkey hash|8 timestamp
var FullIdPubkey = next()

func FullIdPubkeyVars() (
	ser *types.Uint40, fid *types.Id, p *types.PubHash, ca *types.Uint64,
) {
	return new(types.Uint40), new(types.Id), new(types.PubHash), new(types.Uint64)
}
func FullIdPubkeyEnc(
	ser *types.Uint40, fid *types.Id, p *types.PubHash, ca *types.Uint64,
) (enc *T) {
	return New(NewPrefix(FullIdPubkey), ser, fid, p, ca)
}
func FullIdPubkeyDec(
	ser *types.Uint40, fid *types.Id, p *types.PubHash, ca *types.Uint64,
) (enc *T) {
	return New(NewPrefix(), ser, fid, p, ca)
}

// CreatedAt is an index that allows search for the timestamp on the event.
//
//	3 prefix|8 timestamp|5 serial
var CreatedAt = next()

func CreatedAtVars() (ca *types.Uint64, ser *types.Uint40) {
	return new(types.Uint64), new(types.Uint40)
}
func CreatedAtEnc(ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(CreatedAt), ca, ser)
}
func CreatedAtDec(ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), ca, ser)
}

// Kind
//
//	3 prefix|2 kind|8 timestamp|5 serial
var Kind = next()

func KindVars() (ki *types.Uint16, ca *types.Uint64, ser *types.Uint40) {
	return new(types.Uint16), new(types.Uint64), new(types.Uint40)
}
func KindEnc(ki *types.Uint16, ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(Kind), ki, ca, ser)
}
func KindDec(ki *types.Uint16, ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), ki, ca, ser)
}

// Pubkey is a composite index that allows search by pubkey
// filtered by timestamp.
//
//	3 prefix|8 pubkey hash|8 timestamp|5 serial
var Pubkey = next()

func PubkeyVars() (p *types.PubHash, ca *types.Uint64, ser *types.Uint40) {
	return new(types.PubHash), new(types.Uint64), new(types.Uint40)
}
func PubkeyEnc(p *types.PubHash, ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(Pubkey), p, ca, ser)
}
func PubkeyDec(p *types.PubHash, ca *types.Uint64, ser *types.Uint40) (enc *T) {
	return New(NewPrefix(), p, ca, ser)
}

// KindPubkey
//
//	3 prefix|2 kind|8 pubkey hash|8 timestamp|5 serial
var KindPubkey = next()

func KindPubkeyVars() (
	ki *types.Uint16, p *types.PubHash, ca *types.Uint64, ser *types.Uint40,
) {
	return new(types.Uint16), new(types.PubHash), new(types.Uint64), new(types.Uint40)
}
func KindPubkeyEnc(
	ki *types.Uint16, p *types.PubHash, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(KindPubkey), ki, p, ca, ser)
}
func KindPubkeyDec(
	ki *types.Uint16, p *types.PubHash, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, ca, ser)
}

// Tag allows searching for a tag and filter by timestamp.
//
//	3 prefix|1 key letter|8 value hash|8 timestamp|5 serial
var Tag = next()

func TagVars() (
	k *types.Letter, v *types.Ident, ca *types.Uint64, ser *types.Uint40,
) {
	return new(types.Letter), new(types.Ident), new(types.Uint64), new(types.Uint40)
}
func TagEnc(
	k *types.Letter, v *types.Ident, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(Tag), k, v, ca, ser)
}
func TagDec(
	k *types.Letter, v *types.Ident, ca *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), k, v, ca, ser)
}

// TagKind
//
//	3 prefix|1 key letter|8 value hash|2 kind|8 timestamp|5 serial
var TagKind = next()

func TagKindVars() (
	k *types.Letter, v *types.Ident, ki *types.Uint16, ca *types.Uint64,
	ser *types.Uint40,
) {
	return new(types.Letter), new(types.Ident), new(types.Uint16), new(types.Uint64), new(types.Uint40)
}
func TagKindEnc(
	k *types.Letter, v *types.Ident, ki *types.Uint16, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(TagKind), ki, k, v, ca, ser)
}
func TagKindDec(
	k *types.Letter, v *types.Ident, ki *types.Uint16, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, k, v, ca, ser)
}

// TagPubkey allows searching for a pubkey, tag and timestamp.
//
//	3 prefix|1 key letter|8 value hash|8 pubkey hash|8 timestamp|5 serial
var TagPubkey = next()

func TagPubkeyVars() (
	k *types.Letter, v *types.Ident, p *types.PubHash, ca *types.Uint64,
	ser *types.Uint40,
) {
	return new(types.Letter), new(types.Ident), new(types.PubHash), new(types.Uint64), new(types.Uint40)
}
func TagPubkeyEnc(
	k *types.Letter, v *types.Ident, p *types.PubHash, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(TagPubkey), p, k, v, ca, ser)
}
func TagPubkeyDec(
	k *types.Letter, v *types.Ident, p *types.PubHash, ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), p, k, v, ca, ser)
}

// TagKindPubkey
//
//	3 prefix|1 key letter|8 value hash|2 kind|8 pubkey hash|8 bytes timestamp|5 serial
var TagKindPubkey = next()

func TagKindPubkeyVars() (
	k *types.Letter, v *types.Ident, ki *types.Uint16, p *types.PubHash,
	ca *types.Uint64,
	ser *types.Uint40,
) {
	return new(types.Letter), new(types.Ident), new(types.Uint16), new(types.PubHash), new(types.Uint64), new(types.Uint40)
}
func TagKindPubkeyEnc(
	k *types.Letter, v *types.Ident, ki *types.Uint16, p *types.PubHash,
	ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(TagKindPubkey), ki, p, k, v, ca, ser)
}
func TagKindPubkeyDec(
	k *types.Letter, v *types.Ident, ki *types.Uint16, p *types.PubHash,
	ca *types.Uint64,
	ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), ki, p, k, v, ca, ser)
}

// Expiration
//
// 3 prefix|8 timestamp|5 serial
var Expiration = next()

func ExpirationVars() (
	exp *types.Uint64, ser *types.Uint40,
) {
	return new(types.Uint64), new(types.Uint40)
}
func ExpirationEnc(
	exp *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(Expiration), exp, ser)
}
func ExpirationDec(
	exp *types.Uint64, ser *types.Uint40,
) (enc *T) {
	return New(NewPrefix(), exp, ser)
}

// Version
//
// 3 prefix|4 version
var Version = next()

func VersionVars() (
	ver *types.Uint32,
) {
	return new(types.Uint32)
}
func VersionEnc(
	ver *types.Uint32,
) (enc *T) {
	return New(NewPrefix(Version), ver)
}
func VersionDec(
	ver *types.Uint32,
) (enc *T) {
	return New(NewPrefix(), ver)
}
