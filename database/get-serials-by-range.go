package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/chk"
	"orly.dev/database/indexes/types"
)

func (d *D) GetSerialsByRange(idx Range) (sers types.Uint40s, err error) {
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
				sers = append(sers, ser)
			}
			return
		},
	); chk.E(err) {
		return
	}

	return
}
