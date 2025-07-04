package database

import (
	"not.realy.lol/database/indexes"
	"not.realy.lol/database/indexes/types/fullid"
	"not.realy.lol/database/indexes/types/identhash"
	"not.realy.lol/database/indexes/types/idhash"
	. "not.realy.lol/database/indexes/types/number"
	"not.realy.lol/database/indexes/types/pubhash"
	"not.realy.lol/event"
)

// GenerateIndexes creates all the indexes for an event.E instance as defined in keys.go.
// It returns a slice of indexes.T that can be used to store the event in the database.
func GenerateIndexes(ev *event.E, serial uint64) (allIndexes []*indexes.T) {
	// Convert serial to Uint40
	ser := new(Uint40)
	ser.Set(serial)

	// Event index
	eventIndex := indexes.EventEnc(ser)
	allIndexes = append(allIndexes, eventIndex)

	// Id index
	idHash := idhash.New()
	idHash.FromId(ev.Id)
	idIndex := indexes.IdEnc(idHash, ser)
	allIndexes = append(allIndexes, idIndex)

	// IdPubkeyCreatedAt index
	fullID := new(fullid.T)
	fullID.FromId(ev.Id)
	pubHash := new(pubhash.T)
	pubHash.FromPubkey(ev.Pubkey)
	createdAt := new(Uint64)
	createdAt.Set(uint64(ev.CreatedAt.V))
	idPubkeyCreatedAtIndex := indexes.IdPubkeyCreatedAtEnc(
		ser, fullID, pubHash, createdAt,
	)
	allIndexes = append(allIndexes, idPubkeyCreatedAtIndex)

	// CreatedAt index
	createdAtIndex := indexes.CreatedAtEnc(createdAt, ser)
	allIndexes = append(allIndexes, createdAtIndex)

	// PubkeyCreatedAt index
	pubkeyCreatedAtIndex := indexes.PubkeyCreatedAtEnc(pubHash, createdAt, ser)
	allIndexes = append(allIndexes, pubkeyCreatedAtIndex)

	// Process tags for tag-related indexes
	if ev.Tags != nil && ev.Tags.Len() > 0 {
		for _, tag := range ev.Tags.ToSliceOfTags() {
			if tag.Len() >= 2 {
				// Get the key and value from the tag
				keyBytes := tag.B(0)
				valueBytes := tag.B(1)

				// Create identhash for key and value
				keyHash := new(identhash.T)
				keyHash.FromIdent(keyBytes)
				valueHash := new(identhash.T)
				valueHash.FromIdent(valueBytes)

				// PubkeyTagCreatedAt index
				pubkeyTagCreatedAtIndex := indexes.PubkeyTagCreatedAtEnc(
					pubHash, keyHash, valueHash, createdAt, ser,
				)
				allIndexes = append(allIndexes, pubkeyTagCreatedAtIndex)

				// TagCreatedAt index
				tagCreatedAtIndex := indexes.TagCreatedAtEnc(
					keyHash, valueHash, createdAt, ser,
				)
				allIndexes = append(allIndexes, tagCreatedAtIndex)

				// Kind-related tag indexes
				kind := new(Uint16)
				kind.Set(uint16(ev.Kind.K))

				// KindTag index
				kindTagIndex := indexes.KindTagEnc(
					kind, keyHash, valueHash, ser,
				)
				allIndexes = append(allIndexes, kindTagIndex)

				// KindTagCreatedAt index
				kindTagCreatedAtIndex := indexes.KindTagCreatedAtEnc(
					kind, keyHash, valueHash, createdAt, ser,
				)
				allIndexes = append(allIndexes, kindTagCreatedAtIndex)

				// KindPubkeyTagCreatedAt index
				kindPubkeyTagCreatedAtIndex := indexes.KindPubkeyTagCreatedAtEnc(
					kind, pubHash, keyHash, valueHash, createdAt, ser,
				)
				allIndexes = append(allIndexes, kindPubkeyTagCreatedAtIndex)
			}
		}
	}

	// Kind index
	kind := new(Uint16)
	kind.Set(uint16(ev.Kind.K))
	kindIndex := indexes.KindEnc(kind, ser)
	allIndexes = append(allIndexes, kindIndex)

	// KindPubkey index
	kindPubkeyIndex := indexes.KindPubkeyEnc(kind, pubHash, ser)
	allIndexes = append(allIndexes, kindPubkeyIndex)

	// KindCreatedAt index
	kindCreatedAtIndex := indexes.KindCreatedAtEnc(kind, createdAt, ser)
	allIndexes = append(allIndexes, kindCreatedAtIndex)

	// KindPubkeyCreatedAt index
	// Note: The KindPubkeyCreatedAtVars function in keys.go seems to have more parameters than used in KindPubkeyCreatedAtEnc
	// Using the correct parameters based on the function signature
	keyHash := new(identhash.T)
	valueHash := new(identhash.T)
	kindPubkeyCreatedAtIndex := indexes.KindPubkeyCreatedAtEnc(
		kind, pubHash, keyHash, valueHash, createdAt, ser,
	)
	allIndexes = append(allIndexes, kindPubkeyCreatedAtIndex)

	return allIndexes
}
