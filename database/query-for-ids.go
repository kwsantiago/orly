package database

import (
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
	"orly.dev/log"
)

func (d *D) QueryForIds(c context.T, f *filter.F) (
	evs []store.IdTsPk, err error,
) {
	var idxs []Range
	if idxs, err = GetIndexesFromFilter(f); chk.E(err) {
		return
	}
	log.I.S(idxs)
	// for _, idx := range idxs {
	// 	if err = d.View(
	// 		func(txn *badger.Txn) (err error) {
	// 			it :=
	// 			return
	// 		},
	// 	); chk.E(err) {
	// 		return
	// 	}
	// }
	return

}
