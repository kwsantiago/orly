package database

import (
	"orly.dev/chk"
	"orly.dev/codecbuf"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
)

// GetIndexesFromFilter returns encoded indexes based on the given filter.
//
// An error is returned if any input values are invalid during encoding.
//
// The indexes are designed so that only one table needs to be iterated, being a
// complete set of combinations of all fields in the event, thus there is no
// need to decode events until they are to be delivered.
func GetIndexesFromFilter(f *filter.T) (idxs [][]byte, err error) {
	// Id
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
				ser := new(types.Uint40)
				idx := indexes.IdEnc(i, ser)
				buf := codecbuf.Get()
				defer codecbuf.Put(buf)
				if err = idx.MarshalWrite(buf); chk.E(err) {
					return
				}
				bytes := make([]byte, buf.Len())
				copy(bytes, buf.Bytes())
				idxs = append(idxs, bytes)
				return
			}(); chk.E(err) {
				return
			}
		}
		return
	}

	// KindPubkeyTagCreatedAt
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Authors != nil && f.Authors.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 && ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) {
		for _, k := range f.Kinds.ToUint16() {
			for _, author := range f.Authors.ToSliceOfBytes() {
				for _, tag := range f.Tags.ToSliceOfTags() {
					if tag.Len() >= 2 && len(tag.S(0)) == 1 {
						if err = func() (err error) {
							kind := new(types.Uint16)
							kind.Set(k)
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
							ca := new(types.Uint64)
							if f.Since != nil {
								ca.Set(uint64(f.Since.V))
							} else if f.Until != nil {
								ca.Set(uint64(f.Until.V))
							}
							ser := new(types.Uint40)
							idx := indexes.KindPubkeyTagCreatedAtEnc(
								kind, p, key, valueHash, ca, ser,
							)
							buf := codecbuf.Get()
							defer codecbuf.Put(buf)
							if err = idx.MarshalWrite(buf); chk.E(err) {
								return
							}
							bytes := make([]byte, buf.Len())
							copy(bytes, buf.Bytes())
							idxs = append(idxs, bytes)
							return
						}(); chk.E(err) {
							return
						}
					}
				}
			}
		}
		return
	}

	// KindTagCreatedAt
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 && ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) {
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
						ca := new(types.Uint64)
						if f.Since != nil {
							ca.Set(uint64(f.Since.V))
						} else if f.Until != nil {
							ca.Set(uint64(f.Until.V))
						}
						ser := new(types.Uint40)
						idx := indexes.KindTagCreatedAtEnc(
							kind, key, valueHash, ca, ser,
						)
						buf := codecbuf.Get()
						defer codecbuf.Put(buf)
						if err = idx.MarshalWrite(buf); chk.E(err) {
							return
						}
						bytes := make([]byte, buf.Len())
						copy(bytes, buf.Bytes())
						idxs = append(idxs, bytes)
						return
					}(); chk.E(err) {
						return
					}
				}
			}
		}
		return
	}

	// KindPubkeyCreatedAt
	if f.Kinds != nil && f.Kinds.Len() > 0 && f.Authors != nil && f.Authors.Len() > 0 && ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) {
		for _, k := range f.Kinds.ToUint16() {
			for _, author := range f.Authors.ToSliceOfBytes() {
				if err = func() (err error) {
					kind := new(types.Uint16)
					kind.Set(k)
					p := new(types.PubHash)
					if err = p.FromPubkey(author); chk.E(err) {
						return
					}
					ca := new(types.Uint64)
					if f.Since != nil {
						ca.Set(uint64(f.Since.V))
					} else if f.Until != nil {
						ca.Set(uint64(f.Until.V))
					}
					ser := new(types.Uint40)
					idx := indexes.KindPubkeyCreatedAtEnc(kind, p, ca, ser)
					buf := codecbuf.Get()
					defer codecbuf.Put(buf)
					if err = idx.MarshalWrite(buf); chk.E(err) {
						return
					}
					bytes := make([]byte, buf.Len())
					copy(bytes, buf.Bytes())
					idxs = append(idxs, bytes)
					return
				}(); chk.E(err) {
					return
				}
			}
		}
		return
	}

	// PubkeyTagCreatedAt
	if f.Authors != nil && f.Authors.Len() > 0 && f.Tags != nil && f.Tags.Len() > 0 && ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) {
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
						ca := new(types.Uint64)
						if f.Since != nil {
							ca.Set(uint64(f.Since.V))
						} else if f.Until != nil {
							ca.Set(uint64(f.Until.V))
						}
						ser := new(types.Uint40)
						idx := indexes.PubkeyTagCreatedAtEnc(
							p, key, valueHash, ca, ser,
						)
						buf := codecbuf.Get()
						defer codecbuf.Put(buf)
						if err = idx.MarshalWrite(buf); chk.E(err) {
							return
						}
						bytes := make([]byte, buf.Len())
						copy(bytes, buf.Bytes())
						idxs = append(idxs, bytes)
						return
					}(); chk.E(err) {
						return
					}
				}
			}
		}
		return
	}

	// TagCreatedAt
	if f.Tags != nil && f.Tags.Len() > 0 && ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) && (f.Authors == nil || f.Authors.Len() == 0) && (f.Kinds == nil || f.Kinds.Len() == 0) {
		for _, tag := range f.Tags.ToSliceOfTags() {
			if tag.Len() >= 2 && len(tag.S(0)) == 1 {
				if err = func() (err error) {
					keyBytes := tag.B(0)
					valueBytes := tag.B(1)
					key := new(types.Letter)
					key.Set(keyBytes[0])
					valueHash := new(types.Ident)
					valueHash.FromIdent(valueBytes)
					ca := new(types.Uint64)
					if f.Since != nil {
						ca.Set(uint64(f.Since.V))
					} else if f.Until != nil {
						ca.Set(uint64(f.Until.V))
					}
					ser := new(types.Uint40)
					idx := indexes.TagCreatedAtEnc(key, valueHash, ca, ser)
					buf := codecbuf.Get()
					defer codecbuf.Put(buf)
					if err = idx.MarshalWrite(buf); chk.E(err) {
						return
					}
					bytes := make([]byte, buf.Len())
					copy(bytes, buf.Bytes())
					idxs = append(idxs, bytes)
					return
				}(); chk.E(err) {
					return
				}
			}
		}
		return
	}

	// KindCreatedAt
	if f.Kinds != nil && f.Kinds.Len() > 0 && ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) && (f.Authors == nil || f.Authors.Len() == 0) && (f.Tags == nil || f.Tags.Len() == 0) {
		for _, k := range f.Kinds.ToUint16() {
			if err = func() (err error) {
				kind := new(types.Uint16)
				kind.Set(k)
				ca := new(types.Uint64)
				if f.Since != nil {
					ca.Set(uint64(f.Since.V))
				} else if f.Until != nil {
					ca.Set(uint64(f.Until.V))
				}
				ser := new(types.Uint40)
				idx := indexes.KindCreatedAtEnc(kind, ca, ser)
				buf := codecbuf.Get()
				defer codecbuf.Put(buf)
				if err = idx.MarshalWrite(buf); chk.E(err) {
					return
				}
				bytes := make([]byte, buf.Len())
				copy(bytes, buf.Bytes())
				idxs = append(idxs, bytes)
				return
			}(); chk.E(err) {
				return
			}
		}
		return
	}

	// PubkeyCreatedAt
	if f.Authors != nil && f.Authors.Len() > 0 && ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) {
		for _, author := range f.Authors.ToSliceOfBytes() {
			if err = func() (err error) {
				p := new(types.PubHash)
				if err = p.FromPubkey(author); chk.E(err) {
					return
				}
				ca := new(types.Uint64)
				if f.Since != nil {
					ca.Set(uint64(f.Since.V))
				} else if f.Until != nil {
					ca.Set(uint64(f.Until.V))
				}
				ser := new(types.Uint40)
				idx := indexes.PubkeyCreatedAtEnc(p, ca, ser)
				buf := codecbuf.Get()
				defer codecbuf.Put(buf)
				if err = idx.MarshalWrite(buf); chk.E(err) {
					return
				}
				bytes := make([]byte, buf.Len())
				copy(bytes, buf.Bytes())
				idxs = append(idxs, bytes)
				return
			}(); chk.E(err) {
				return
			}
		}
		return
	}

	// CreatedAt
	if ((f.Since != nil && f.Since.V != 0) || (f.Until != nil && f.Until.V != 0)) && (f.Authors == nil || f.Authors.Len() == 0) && (f.Kinds == nil || f.Kinds.Len() == 0) && (f.Tags == nil || f.Tags.Len() == 0) {
		if err = func() (err error) {
			ca := new(types.Uint64)
			if f.Since != nil {
				ca.Set(uint64(f.Since.V))
			} else if f.Until != nil {
				ca.Set(uint64(f.Until.V))
			}
			ser := new(types.Uint40)
			idx := indexes.CreatedAtEnc(ca, ser)
			buf := codecbuf.Get()
			defer codecbuf.Put(buf)
			if err = idx.MarshalWrite(buf); chk.E(err) {
				return
			}
			bytes := make([]byte, buf.Len())
			copy(bytes, buf.Bytes())
			idxs = append(idxs, bytes)
			return
		}(); chk.E(err) {
			return
		}
		return
	}

	return
}
