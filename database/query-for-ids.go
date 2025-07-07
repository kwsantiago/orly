package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/database/indexes/types"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
	"sort"
)

func (d *D) QueryForIds(c context.T, f *filter.F) (
	evs []store.IdPkTs, err error,
) {
	var idxs []Range
	if idxs, err = GetIndexesFromFilter(f); chk.E(err) {
		return
	}

	for _, idx := range idxs {
		// Id searches are a special case as they don't require iteration
		if bytes.Equal(idx.Start, idx.End) {
			// this is an Id search
			var ser *types.Uint40
			if ser, err = d.GetSerialById(idx.Start); chk.E(err) {
				return
			}
			// scan for the IdPkTs
			var fidpk *store.IdPkTs
			if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
				return
			}
			evs = append(evs, *fidpk)
		} else {
			var founds types.Uint40s
			if err = d.View(
				func(txn *badger.Txn) (err error) {
					it := txn.NewIterator(
						badger.IteratorOptions{
							Reverse: true,
						},
					)
					defer it.Close()
					var count int
					for it.Seek(idx.End); it.Valid(); it.Next() {
						count++
						item := it.Item()
						var key []byte
						key = item.KeyCopy(nil)
						if bytes.Compare(
							key[:len(key)-5],
							idx.Start,
						) < 0 {
							// didn't find it within the timestamp range
							return
						}

						ser := new(types.Uint40)
						buf := bytes.NewBuffer(key[len(key)-5:])
						if err = ser.UnmarshalRead(buf); chk.E(err) {
							return
						}
						founds = append(founds, ser)
					}
					return
				},
			); chk.E(err) {
				return
			}
			// fetch the events
			for _, ser := range founds {
				// scan for the IdPkTs
				var fidpk *store.IdPkTs
				if fidpk, err = d.GetFullIdPubkeyBySerial(ser); chk.E(err) {
					return
				}
				evs = append(evs, *fidpk)
			}
			// sort results by timestamp in reverse chronological order
			sort.Slice(
				evs, func(i, j int) bool {
					return evs[i].Ts > evs[j].Ts
				},
			)
		}
	}
	return

}
