package database

import (
	"not.realy.lol/database/indexes"
	"not.realy.lol/database/indexes/types"
	"not.realy.lol/event"
)

// GetIndexesForEvent creates all the indexes for an event.E instance as defined in
// keys.go. It returns a slice of indexes.T that can be used to store the event
// in the database.
func GetIndexesForEvent(ev *event.E, serial uint64) (idxs []*indexes.T) {
	// Convert serial to Uint40
	ser := new(types.Uint40)
	ser.Set(serial)

	// Id index
	idHash := new(types.IdHash)
	idHash.FromId(ev.Id)
	idIndex := indexes.IdEnc(idHash, ser)
	idxs = append(idxs, idIndex)

	// IdPubkeyCreatedAt index
	fullID := new(types.Id)
	fullID.FromId(ev.Id)
	pubHash := new(types.PubHash)
	pubHash.FromPubkey(ev.Pubkey)
	createdAt := new(types.Uint64)
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
			if tag.Len() >= 2 && len(tag.S(0)) == 1 {
				// Get the key and value from the tag
				keyBytes := tag.B(0)
				valueBytes := tag.B(1)

				// Create identhash for key and value
				keyHash := new(types.Letter)
				keyHash.Set(keyBytes[0])
				valueHash := new(types.Ident)
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
				kind := new(types.Uint16)
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
	kind := new(types.Uint16)
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
	// Note: The KindPubkeyCreatedAtVars function in keys.go seems to have more parameters than used in KindPubkeyCreatedAtEnc
	// Using the correct parameters based on the function signature
	keyHash := new(types.Ident)
	valueHash := new(types.Ident)
	kindPubkeyCreatedAtIndex := indexes.KindPubkeyCreatedAtEnc(
		kind, pubHash, keyHash, valueHash, createdAt, ser,
	)
	idxs = append(idxs, kindPubkeyCreatedAtIndex)

	return idxs
}
