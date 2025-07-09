package database

import (
	"bytes"
	"math"
	"orly.dev/chk"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
	"orly.dev/log"
)

type Range struct {
	Start, End []byte
}

// GetIndexesFromFilter returns encoded indexes based on the given filter.
//
// An error is returned if any input values are invalid during encoding.
//
// The indexes are designed so that only one table needs to be iterated, being a
// complete set of combinations of all fields in the event, thus there is no
// need to decode events until they are to be delivered.
func GetIndexesFromFilter(f *filter.F) (idxs []Range, err error) {
	log.T.F("getting range indexes for filter: %s", f.Serialize())
	// Id eid
	//
	// If there is any Ids in the filter, none of the other fields matter. It
	// should be an error, but convention just ignores it.
	if f.Ids.Len() > 0 {
		for _, id := range f.Ids.ToSliceOfBytes() {
			if err = func() (err error) {
				i := new(types.IdHash)
				if err = i.FromId(id); chk.E(err) {
					return
				}
				buf := new(bytes.Buffer)
				idx := indexes.IdEnc(i, nil)
				if err = idx.MarshalWrite(buf); chk.E(err) {
					return
				}
				b := buf.Bytes()
				r := Range{b, b}
				idxs = append(idxs, r)
				return
			}(); chk.E(err) {
				return
			}
		}
		return
	}

	caStart := new(types.Uint64)
	caEnd := new(types.Uint64)

	// Set the start of range (Since or default to zero)
	if f.Since != nil && f.Since.V != 0 {
		caStart.Set(uint64(f.Since.V))
	} else {
		caStart.Set(uint64(0))
	}

	// Set the end of range (Until or default to math.MaxInt64)
	if f.Until != nil && f.Until.V != 0 {
		caEnd.Set(uint64(f.Until.V + 1))
	} else {
		caEnd.Set(uint64(math.MaxInt64))
	}

	// KindPubkeyTag kpt
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Authors != nil && f.Authors.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 {
		log.T.F("kinds authors tags")
		for _, k := range f.Kinds.ToUint16() {
			for _, author := range f.Authors.ToSliceOfBytes() {
				for _, tag := range f.Tags.ToSliceOfTags() {
					log.I.S(tag)
					if tag.Len() >= 2 && len(tag.S(0)) == 2 {
						kind := new(types.Uint16)
						kind.Set(k)
						p := new(types.PubHash)
						if err = p.FromPubkey(author); chk.E(err) {
							return
						}
						keyBytes := tag.B(0)[1:]
						for _, valueBytes := range tag.ToSliceOfBytes()[1:] {
							key := new(types.Letter)
							key.Set(keyBytes[0])
							valueHash := new(types.Ident)
							valueHash.FromIdent(valueBytes)
							start, end := new(bytes.Buffer), new(bytes.Buffer)
							idxS := indexes.KindPubkeyTagEnc(
								kind, p, key, valueHash, caStart, nil,
							)
							if err = idxS.MarshalWrite(start); chk.E(err) {
								return
							}
							idxE := indexes.KindPubkeyTagEnc(
								kind, p, key, valueHash, caEnd, nil,
							)
							if err = idxE.MarshalWrite(end); chk.E(err) {
								return
							}
							idxs = append(
								idxs, Range{
									start.Bytes(), end.Bytes(),
								},
							)
						}
						return
					}
				}
			}
		}
		return
	}

	// KindTag ktc
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 {
		for _, k := range f.Kinds.ToUint16() {
			for _, tag := range f.Tags.ToSliceOfTags() {
				if tag.Len() >= 2 && len(tag.S(0)) == 1 {
					if err = func() (err error) {
						kind := new(types.Uint16)
						kind.Set(k)
						keyBytes := tag.B(0)
						valueBytes := tag.B(1)
						key := new(types.Letter)
						key.Set(keyBytes[0])
						valueHash := new(types.Ident)
						valueHash.FromIdent(valueBytes)
						start, end := new(bytes.Buffer), new(bytes.Buffer)
						idxS := indexes.KindTagEnc(
							kind, key, valueHash, caStart, nil,
						)
						if err = idxS.MarshalWrite(start); chk.E(err) {
							return
						}
						idxE := indexes.KindTagEnc(
							kind, key, valueHash, caEnd, nil,
						)
						if err = idxE.MarshalWrite(end); chk.E(err) {
							return
						}
						idxs = append(
							idxs, Range{
								start.Bytes(), end.Bytes(),
							},
						)
						return
					}(); chk.E(err) {
						return
					}
				}
			}
		}
		return
	}

	// KindPubkey kpc
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Authors != nil && f.Authors.Len() > 0 {
		for _, k := range f.Kinds.ToUint16() {
			for _, author := range f.Authors.ToSliceOfBytes() {
				if err = func() (err error) {
					kind := new(types.Uint16)
					kind.Set(k)
					p := new(types.PubHash)
					if err = p.FromPubkey(author); chk.E(err) {
						return
					}
					start, end := new(bytes.Buffer), new(bytes.Buffer)
					idxS := indexes.KindPubkeyEnc(kind, p, caStart, nil)
					if err = idxS.MarshalWrite(start); chk.E(err) {
						return
					}
					idxE := indexes.KindPubkeyEnc(kind, p, caEnd, nil)
					if err = idxE.MarshalWrite(end); chk.E(err) {
						return
					}
					idxs = append(
						idxs, Range{start.Bytes(), end.Bytes()},
					)
					return
				}(); chk.E(err) {
					return
				}
			}
		}
		return
	}

	// PubkeyTag ptc
	if f.Authors != nil && f.Authors.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 {
		for _, author := range f.Authors.ToSliceOfBytes() {
			for _, tag := range f.Tags.ToSliceOfTags() {
				if tag.Len() >= 2 && len(tag.S(0)) == 1 {
					if err = func() (err error) {
						p := new(types.PubHash)
						if err = p.FromPubkey(author); chk.E(err) {
							return
						}
						keyBytes := tag.B(0)
						valueBytes := tag.B(1)
						key := new(types.Letter)
						key.Set(keyBytes[0])
						valueHash := new(types.Ident)
						valueHash.FromIdent(valueBytes)
						start, end := new(bytes.Buffer), new(bytes.Buffer)
						idxS := indexes.PubkeyTagEnc(
							p, key, valueHash, caStart, nil,
						)
						if err = idxS.MarshalWrite(start); chk.E(err) {
							return
						}
						idxE := indexes.PubkeyTagEnc(
							p, key, valueHash, caEnd, nil,
						)
						if err = idxE.MarshalWrite(end); chk.E(err) {
							return
						}
						idxs = append(
							idxs, Range{start.Bytes(), end.Bytes()},
						)
						return
					}(); chk.E(err) {
						return
					}
				}
			}
		}
		return
	}

	// Tag  itc
	if f.Tags != nil && f.Tags.Len() > 0 && (f.Authors == nil || f.Authors.Len() == 0) && (f.Kinds == nil || f.Kinds.Len() == 0) {
		for _, tag := range f.Tags.ToSliceOfTags() {
			if tag.Len() >= 2 && len(tag.S(0)) == 1 {
				if err = func() (err error) {
					keyBytes := tag.B(0)
					valueBytes := tag.B(1)
					key := new(types.Letter)
					key.Set(keyBytes[0])
					valueHash := new(types.Ident)
					valueHash.FromIdent(valueBytes)
					start, end := new(bytes.Buffer), new(bytes.Buffer)
					idxS := indexes.TagEnc(key, valueHash, caStart, nil)
					if err = idxS.MarshalWrite(start); chk.E(err) {
						return
					}
					idxE := indexes.TagEnc(key, valueHash, caEnd, nil)
					if err = idxE.MarshalWrite(end); chk.E(err) {
						return
					}
					idxs = append(
						idxs, Range{start.Bytes(), end.Bytes()},
					)
					return
				}(); chk.E(err) {
					return
				}
			}
		}
		return
	}

	// Kind kca
	if f.Kinds != nil && f.Kinds.Len() > 0 && (f.Authors == nil || f.Authors.Len() == 0) && (f.Tags == nil || f.Tags.Len() == 0) {
		for _, k := range f.Kinds.ToUint16() {
			if err = func() (err error) {
				kind := new(types.Uint16)
				kind.Set(k)
				start, end := new(bytes.Buffer), new(bytes.Buffer)
				idxS := indexes.KindEnc(kind, caStart, nil)
				if err = idxS.MarshalWrite(start); chk.E(err) {
					return
				}
				idxE := indexes.KindEnc(kind, caEnd, nil)
				if err = idxE.MarshalWrite(end); chk.E(err) {
					return
				}
				idxs = append(
					idxs, Range{start.Bytes(), end.Bytes()},
				)
				return
			}(); chk.E(err) {
				return
			}
		}
		return
	}

	// Pubkey pca
	if f.Authors != nil && f.Authors.Len() > 0 {
		for _, author := range f.Authors.ToSliceOfBytes() {
			if err = func() (err error) {
				p := new(types.PubHash)
				if err = p.FromPubkey(author); chk.E(err) {
					return
				}
				start, end := new(bytes.Buffer), new(bytes.Buffer)
				idxS := indexes.PubkeyEnc(p, caStart, nil)
				if err = idxS.MarshalWrite(start); chk.E(err) {
					return
				}
				idxE := indexes.PubkeyEnc(p, caEnd, nil)
				if err = idxE.MarshalWrite(end); chk.E(err) {
					return
				}
				idxs = append(
					idxs, Range{start.Bytes(), end.Bytes()},
				)
				return
			}(); chk.E(err) {
				return
			}
		}
		return
	}

	// CreatedAt ica
	start, end := new(bytes.Buffer), new(bytes.Buffer)
	idxS := indexes.CreatedAtEnc(caStart, nil)
	if err = idxS.MarshalWrite(start); chk.E(err) {
		return
	}
	idxE := indexes.CreatedAtEnc(caEnd, nil)
	if err = idxE.MarshalWrite(end); chk.E(err) {
		return
	}
	idxs = append(
		idxs, Range{start.Bytes(), end.Bytes()},
	)
	return
}
