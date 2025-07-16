package database

import (
	"bytes"
	"math"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/encoders/filter"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
	"sort"
)

type Range struct {
	Start, End []byte
}

// isHexString checks if the byte slice contains only hex characters
func isHexString(data []byte) (isHex bool) {
	if len(data)%2 != 0 {
		return false
	}
	for _, b := range data {
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')) {
			return false
		}
	}
	return true
}

// createIdHashFromData creates an IdHash from data that could be hex or binary
func createIdHashFromData(data []byte) (i *types.IdHash, err error) {
	i = new(types.IdHash)

	// If data looks like hex string and has the right length for hex-encoded
	// sha256
	if len(data) == 64 && isHexString(data) {
		if err = i.FromIdHex(string(data)); chk.E(err) {
			return
		}
	} else {
		// Assume it's binary data
		if err = i.FromId(data); chk.E(err) {
			return
		}
	}
	return
}

// createPubHashFromData creates a PubHash from data that could be hex or binary
func createPubHashFromData(data []byte) (p *types.PubHash, err error) {
	p = new(types.PubHash)

	// If data looks like hex string and has the right length for hex-encoded
	// pubkey
	if len(data) == 64 && isHexString(data) {
		if err = p.FromPubkeyHex(string(data)); chk.E(err) {
			return
		}
	} else {
		// Assume it's binary data
		if err = p.FromPubkey(data); chk.E(err) {
			return
		}
	}
	return
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
				var i *types.IdHash
				if i, err = createIdHashFromData(id); chk.E(err) {
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

	if f.Tags != nil && f.Tags.Len() > 1 {
		// sort the tags so they are in iteration order (reverse)
		tmp := f.Tags.ToSliceOfTags()
		sort.Slice(
			tmp, func(i, j int) bool {
				return bytes.Compare(tmp[i].B(0), tmp[j].B(0)) > 0
			},
		)
	}

	// TagKindPubkey tkp
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Authors != nil && f.Authors.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 {
		for _, k := range f.Kinds.ToUint16() {
			for _, author := range f.Authors.ToSliceOfBytes() {
				for _, tag := range f.Tags.ToSliceOfTags() {
					if tag.Len() >= 2 && (len(tag.S(0)) == 1 || (len(tag.S(0)) == 2 && tag.S(0)[0] == '#')) {
						kind := new(types.Uint16)
						kind.Set(k)
						var p *types.PubHash
						if p, err = createPubHashFromData(author); chk.E(err) {
							return
						}
						keyBytes := tag.B(0)
						key := new(types.Letter)
						// If the tag key starts with '#', use the second character as the key
						if len(keyBytes) == 2 && keyBytes[0] == '#' {
							key.Set(keyBytes[1])
						} else {
							key.Set(keyBytes[0])
						}
						for _, valueBytes := range tag.ToSliceOfBytes()[1:] {
							valueHash := new(types.Ident)
							valueHash.FromIdent(valueBytes)
							start, end := new(bytes.Buffer), new(bytes.Buffer)
							idxS := indexes.TagKindPubkeyEnc(
								key, valueHash, kind, p, caStart, nil,
							)
							if err = idxS.MarshalWrite(start); chk.E(err) {
								return
							}
							idxE := indexes.TagKindPubkeyEnc(
								key, valueHash, kind, p, caEnd, nil,
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
					}
				}
			}
		}
		return
	}

	// TagKind tkc
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 {
		for _, k := range f.Kinds.ToUint16() {
			for _, tag := range f.Tags.ToSliceOfTags() {
				if tag.Len() >= 2 && (len(tag.S(0)) == 1 || (len(tag.S(0)) == 2 && tag.S(0)[0] == '#')) {
					kind := new(types.Uint16)
					kind.Set(k)
					keyBytes := tag.B(0)
					key := new(types.Letter)
					// If the tag key starts with '#', use the second character as the key
					if len(keyBytes) == 2 && keyBytes[0] == '#' {
						key.Set(keyBytes[1])
					} else {
						key.Set(keyBytes[0])
					}
					for _, valueBytes := range tag.ToSliceOfBytes()[1:] {
						valueHash := new(types.Ident)
						valueHash.FromIdent(valueBytes)
						start, end := new(bytes.Buffer), new(bytes.Buffer)
						idxS := indexes.TagKindEnc(
							key, valueHash, kind, caStart, nil,
						)
						if err = idxS.MarshalWrite(start); chk.E(err) {
							return
						}
						idxE := indexes.TagKindEnc(
							key, valueHash, kind, caEnd, nil,
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
				}
			}
		}
		return
	}

	// TagPubkey tpc
	if f.Authors != nil && f.Authors.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 {
		for _, author := range f.Authors.ToSliceOfBytes() {
			for _, tag := range f.Tags.ToSliceOfTags() {
				if tag.Len() >= 2 && (len(tag.S(0)) == 1 || (len(tag.S(0)) == 2 && tag.S(0)[0] == '#')) {
					var p *types.PubHash
					if p, err = createPubHashFromData(author); chk.E(err) {
						return
					}
					keyBytes := tag.B(0)
					key := new(types.Letter)
					// If the tag key starts with '#', use the second character as the key
					if len(keyBytes) == 2 && keyBytes[0] == '#' {
						key.Set(keyBytes[1])
					} else {
						key.Set(keyBytes[0])
					}
					for _, valueBytes := range tag.ToSliceOfBytes()[1:] {
						valueHash := new(types.Ident)
						valueHash.FromIdent(valueBytes)
						start, end := new(bytes.Buffer), new(bytes.Buffer)
						idxS := indexes.TagPubkeyEnc(
							key, valueHash, p, caStart, nil,
						)
						if err = idxS.MarshalWrite(start); chk.E(err) {
							return
						}
						idxE := indexes.TagPubkeyEnc(
							key, valueHash, p, caEnd, nil,
						)
						if err = idxE.MarshalWrite(end); chk.E(err) {
							return
						}
						idxs = append(
							idxs, Range{start.Bytes(), end.Bytes()},
						)
					}
				}
			}
		}
		return
	}

	// Tag tc-
	if f.Tags != nil && f.Tags.Len() > 0 && (f.Authors == nil || f.Authors.Len() == 0) && (f.Kinds == nil || f.Kinds.Len() == 0) {
		for _, tag := range f.Tags.ToSliceOfTags() {
			if tag.Len() >= 2 && (len(tag.S(0)) == 1 || (len(tag.S(0)) == 2 && tag.S(0)[0] == '#')) {
				keyBytes := tag.B(0)
				key := new(types.Letter)
				// If the tag key starts with '#', use the second character as the key
				if len(keyBytes) == 2 && keyBytes[0] == '#' {
					key.Set(keyBytes[1])
				} else {
					key.Set(keyBytes[0])
				}
				for _, valueBytes := range tag.ToSliceOfBytes()[1:] {
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
				}
			}
		}
		return
	}

	// KindPubkey kpc
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Authors != nil && f.Authors.Len() > 0 {
		for _, k := range f.Kinds.ToUint16() {
			for _, author := range f.Authors.ToSliceOfBytes() {
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
			}
		}
		return
	}

	// Kind kc-
	if f.Kinds != nil && f.Kinds.Len() > 0 && (f.Authors == nil || f.Authors.Len() == 0) && (f.Tags == nil || f.Tags.Len() == 0) {
		for _, k := range f.Kinds.ToUint16() {
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
		}
		return
	}

	// Pubkey pc-
	if f.Authors != nil && f.Authors.Len() > 0 {
		for _, author := range f.Authors.ToSliceOfBytes() {
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
		}
		return
	}

	// CreatedAt c--
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
