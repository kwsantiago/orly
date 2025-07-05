package database

import (
	"orly.dev/chk"
	"orly.dev/codecbuf"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
)

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
				buf := codecbuf.Get()
				defer codecbuf.Put(buf)
				err = i.MarshalWrite(buf)
				return
			}(); chk.E(err) {
				return
			}
		}
		return
	}
	// PubkeyCreatedAt

	// CreatedAt

	// PubkeyTagCreatedAt

	// TagCreatedAt

	// Kind

	// KindCreatedAt

	// KindPubkey

	// KindPubkeyCreatedAt

	// KindTag

	// KindTagCreatedAt

	// KindPubkeyTagCreatedAt

	return
}
