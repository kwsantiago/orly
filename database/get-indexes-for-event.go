package database

import (
	"orly.dev/chk"
	"orly.dev/database/indexes"
	. "orly.dev/database/indexes/types"
	"orly.dev/event"
)

// GetIndexesForEvent creates all the indexes for an event.E instance as defined
// in keys.go. It returns a slice of indexes.T that can be used to store the
// event in the database.
func GetIndexesForEvent(ev *event.E, serial uint64) (
	idxs []*indexes.T, err error,
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
	idxs = append(idxs, idIndex)
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
	idxs = append(idxs, idPubkeyCreatedAtIndex)
	// CreatedAt index
	createdAtIndex := indexes.CreatedAtEnc(createdAt, ser)
	idxs = append(idxs, createdAtIndex)
	// PubkeyCreatedAt index
	pubkeyCreatedAtIndex := indexes.PubkeyCreatedAtEnc(pubHash, createdAt, ser)
	idxs = append(idxs, pubkeyCreatedAtIndex)
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
				idxs = append(idxs, pubkeyTagCreatedAtIndex)
				// TagCreatedAt index
				tagCreatedAtIndex := indexes.TagCreatedAtEnc(
					keyHash, valueHash, createdAt, ser,
				)
				idxs = append(idxs, tagCreatedAtIndex)
				// Kind-related tag indexes
				kind := new(Uint16)
				kind.Set(uint16(ev.Kind.K))
				// KindTag index
				kindTagIndex := indexes.KindTagEnc(
					kind, keyHash, valueHash, ser,
				)
				idxs = append(idxs, kindTagIndex)
				// KindTagCreatedAt index
				kindTagCreatedAtIndex := indexes.KindTagCreatedAtEnc(
					kind, keyHash, valueHash, createdAt, ser,
				)
				idxs = append(idxs, kindTagCreatedAtIndex)
				// KindPubkeyTagCreatedAt index
				kindPubkeyTagCreatedAtIndex := indexes.KindPubkeyTagCreatedAtEnc(
					kind, pubHash, keyHash, valueHash, createdAt, ser,
				)
				idxs = append(idxs, kindPubkeyTagCreatedAtIndex)
			}
		}
	}
	// Kind index
	kind := new(Uint16)
	kind.Set(uint16(ev.Kind.K))
	kindIndex := indexes.KindEnc(kind, ser)
	idxs = append(idxs, kindIndex)
	// KindPubkey index
	kindPubkeyIndex := indexes.KindPubkeyEnc(kind, pubHash, ser)
	idxs = append(idxs, kindPubkeyIndex)
	// KindCreatedAt index
	kindCreatedAtIndex := indexes.KindCreatedAtEnc(kind, createdAt, ser)
	idxs = append(idxs, kindCreatedAtIndex)
	// KindPubkeyCreatedAt index
	// Using the correct parameters based on the function signature
	kindPubkeyCreatedAtIndex := indexes.KindPubkeyCreatedAtEnc(
		kind, pubHash, createdAt, ser,
	)
	idxs = append(idxs, kindPubkeyCreatedAtIndex)
	return
}
