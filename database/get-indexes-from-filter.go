package database

import (
	"not.realy.lol/chk"
	"not.realy.lol/codecbuf"
	"not.realy.lol/database/indexes"
	"not.realy.lol/database/indexes/types/idhash"
	"not.realy.lol/filter"
)

func GetIndexesFromFilter(f *filter.T) (idxs [][]byte, err error) {
	// Id
	//
	// If there is any Ids in the filter, none of the other fields matter. It
	// should be an error but convention just ignores it.
	if f.Ids.Len() > 0 {
		for _, id := range f.Ids.ToSliceOfBytes() {
			if err = func() (err error) {
				i := idhash.New()
				if err = i.FromId(id); chk.E(err) {
					return
				}
				buf := codecbuf.Get()
				defer codecbuf.Put(buf)
				err = indexes.IdSearch(i).MarshalWrite(buf)
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
