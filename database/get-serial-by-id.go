package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/database/indexes/types"
	"orly.dev/utils/chk"
)

func (d *D) GetSerialById(idx []byte) (ser *types.Uint40, err error) {
	if err = d.View(
		func(txn *badger.Txn) (err error) {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			var key []byte
			defer it.Close()
			it.Seek(idx)
			if it.ValidForPrefix(idx) {
				item := it.Item()
				key = item.KeyCopy(nil)
				ser = new(types.Uint40)
				buf := bytes.NewBuffer(key[len(key)-5:])
				if err = ser.UnmarshalRead(buf); chk.E(err) {
					return
				}
			} else {
				// just don't return what we don't have? others may be
				// found tho.
			}
			return
		},
	); chk.E(err) {
		return
	}
	return
}
