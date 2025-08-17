package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/utils/chk"
	"sort"
)

func (d *D) GetSerialsByRange(idx Range) (
	sers types.Uint40s, err error,
) {
	if err = d.View(
		func(txn *badger.Txn) (err error) {
			it := txn.NewIterator(
				badger.IteratorOptions{
					Reverse: true,
				},
			)
			defer it.Close()
			for it.Seek(idx.End); it.Valid(); it.Next() {
				item := it.Item()
				var key []byte
				key = item.Key()
				if bytes.Compare(
					key[:len(key)-5], idx.Start,
				) < 0 {
					// didn't find it within the timestamp range
					return
				}
				ser := new(types.Uint40)
				buf := bytes.NewBuffer(key[len(key)-5:])
				if err = ser.UnmarshalRead(buf); chk.E(err) {
					return
				}
				sers = append(sers, ser)
			}
			return
		},
	); chk.E(err) {
		return
	}
	sort.Slice(
		sers, func(i, j int) bool {
			return sers[i].Get() < sers[j].Get()
		},
	)
	return
}
