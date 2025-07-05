package database

import (
	"orly.dev/chk"
	"orly.dev/codecbuf"
	"orly.dev/database/indexes"
	. "orly.dev/database/indexes/types"
	"orly.dev/event"
)

// appendIndexBytes marshals an index to a byte slice and appends it to the idxs slice
func appendIndexBytes(idxs *[][]byte, idx *indexes.T) (err error) {
	// Get a buffer from the pool
	buf := codecbuf.Get()
	defer codecbuf.Put(buf)
	// Ensure the buffer is empty
	buf.Reset()
	// Marshal the index to the buffer
	if err = idx.MarshalWrite(buf); chk.E(err) {
		return
	}
	// Copy the buffer's bytes to a new byte slice
	bytes := make([]byte, buf.Len())
	copy(bytes, buf.Bytes())
	// Append the byte slice to the idxs slice
	*idxs = append(*idxs, bytes)
	return
}

// GetIndexesForEvent creates all the indexes for an event.E instance as defined
// in keys.go. It returns a slice of byte slices that can be used to store the
// event in the database.
func GetIndexesForEvent(ev *event.E, serial uint64) (
	idxs [][]byte, err error,
) {
	defer func() {
		if err != nil {
			idxs = nil
		}
	}()
	// Convert serial to Uint40
	ser := new(Uint40)
	if err = ser.Set(serial); chk.E(err) {
		return
	}
	// Id index
	idHash := new(IdHash)
	if err = idHash.FromId(ev.Id); chk.E(err) {
		return
	}
	idIndex := indexes.IdEnc(idHash, ser)
	if err = appendIndexBytes(&idxs, idIndex); chk.E(err) {
		return
	}
	// IdPubkeyCreatedAt index
	fullID := new(Id)
	if err = fullID.FromId(ev.Id); chk.E(err) {
		return
	}
	pubHash := new(PubHash)
	if err = pubHash.FromPubkey(ev.Pubkey); chk.E(err) {
		return
	}
	createdAt := new(Uint64)
	createdAt.Set(uint64(ev.CreatedAt.V))
	idPubkeyCreatedAtIndex := indexes.IdPubkeyCreatedAtEnc(
		ser, fullID, pubHash, createdAt,
	)
	if err = appendIndexBytes(&idxs, idPubkeyCreatedAtIndex); chk.E(err) {
		return
	}
	// CreatedAt index
	createdAtIndex := indexes.CreatedAtEnc(createdAt, ser)
	if err = appendIndexBytes(&idxs, createdAtIndex); chk.E(err) {
		return
	}
	// PubkeyCreatedAt index
	pubkeyCreatedAtIndex := indexes.PubkeyCreatedAtEnc(pubHash, createdAt, ser)
	if err = appendIndexBytes(&idxs, pubkeyCreatedAtIndex); chk.E(err) {
		return
	}
	// Process tags for tag-related indexes
	if ev.Tags != nil && ev.Tags.Len() > 0 {
		for _, tag := range ev.Tags.ToSliceOfTags() {
			// only index tags with a value field and the key is a single
			// character
			if tag.Len() >= 2 && len(tag.S(0)) == 1 {
				// Get the key and value from the tag
				keyBytes := tag.B(0)
				// if the key is not a-zA-Z skip
				if (keyBytes[0] < 'a' && keyBytes[0] > 'z') || (keyBytes[0] < 'A' && keyBytes[0] > 'Z') {
					continue
				}
				valueBytes := tag.B(1)
				// Create identhash for key and value
				keyHash := new(Letter)
				keyHash.Set(keyBytes[0])
				valueHash := new(Ident)
				valueHash.FromIdent(valueBytes)
				// PubkeyTagCreatedAt index
				pubkeyTagCreatedAtIndex := indexes.PubkeyTagCreatedAtEnc(
					pubHash, keyHash, valueHash, createdAt, ser,
				)
				if err = appendIndexBytes(&idxs, pubkeyTagCreatedAtIndex); chk.E(err) {
					return
				}
				// TagCreatedAt index
				tagCreatedAtIndex := indexes.TagCreatedAtEnc(
					keyHash, valueHash, createdAt, ser,
				)
				if err = appendIndexBytes(&idxs, tagCreatedAtIndex); chk.E(err) {
					return
				}
				// Kind-related tag indexes
				kind := new(Uint16)
				kind.Set(uint16(ev.Kind.K))
				// KindTag index
				kindTagIndex := indexes.KindTagEnc(
					kind, keyHash, valueHash, ser,
				)
				if err = appendIndexBytes(&idxs, kindTagIndex); chk.E(err) {
					return
				}
				// KindTagCreatedAt index
				kindTagCreatedAtIndex := indexes.KindTagCreatedAtEnc(
					kind, keyHash, valueHash, createdAt, ser,
				)
				if err = appendIndexBytes(&idxs, kindTagCreatedAtIndex); chk.E(err) {
					return
				}
				// KindPubkeyTagCreatedAt index
				kindPubkeyTagCreatedAtIndex := indexes.KindPubkeyTagCreatedAtEnc(
					kind, pubHash, keyHash, valueHash, createdAt, ser,
				)
				if err = appendIndexBytes(&idxs, kindPubkeyTagCreatedAtIndex); chk.E(err) {
					return
				}
			}
		}
	}
	// Kind index
	kind := new(Uint16)
	kind.Set(uint16(ev.Kind.K))
	kindIndex := indexes.KindEnc(kind, ser)
	if err = appendIndexBytes(&idxs, kindIndex); chk.E(err) {
		return
	}
	// KindPubkey index
	kindPubkeyIndex := indexes.KindPubkeyEnc(kind, pubHash, ser)
	if err = appendIndexBytes(&idxs, kindPubkeyIndex); chk.E(err) {
		return
	}
	// KindCreatedAt index
	kindCreatedAtIndex := indexes.KindCreatedAtEnc(kind, createdAt, ser)
	if err = appendIndexBytes(&idxs, kindCreatedAtIndex); chk.E(err) {
		return
	}
	// KindPubkeyCreatedAt index
	// Using the correct parameters based on the function signature
	kindPubkeyCreatedAtIndex := indexes.KindPubkeyCreatedAtEnc(
		kind, pubHash, createdAt, ser,
	)
	if err = appendIndexBytes(&idxs, kindPubkeyCreatedAtIndex); chk.E(err) {
		return
	}
	return
}
