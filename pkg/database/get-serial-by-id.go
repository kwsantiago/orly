package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/errorf"
)

func (d *D) GetSerialById(id []byte) (ser *types.Uint40, err error) {
	var idxs []Range
	if idxs, err = GetIndexesFromFilter(&filter.F{Ids: tag.New(id)}); chk.E(err) {
		return
	}
	if len(idxs) == 0 {
		err = errorf.E("no indexes found for id %0x", id)
	}
	if err = d.View(
		func(txn *badger.Txn) (err error) {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			var key []byte
			defer it.Close()
			it.Seek(idxs[0].Start)
			if it.ValidForPrefix(idxs[0].Start) {
				item := it.Item()
				key = item.Key()
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

//
// func (d *D) GetSerialBytesById(id []byte) (ser []byte, err error) {
// 	var idxs []Range
// 	if idxs, err = GetIndexesFromFilter(&filter.F{Ids: tag.New(id)}); chk.E(err) {
// 		return
// 	}
// 	if len(idxs) == 0 {
// 		err = errorf.E("no indexes found for id %0x", id)
// 	}
// 	if err = d.View(
// 		func(txn *badger.Txn) (err error) {
// 			it := txn.NewIterator(badger.DefaultIteratorOptions)
// 			var key []byte
// 			defer it.Close()
// 			it.Seek(idxs[0].Start)
// 			if it.ValidForPrefix(idxs[0].Start) {
// 				item := it.Item()
// 				key = item.Key()
// 				ser = key[len(key)-5:]
// 			} else {
// 				// just don't return what we don't have? others may be
// 				// found tho.
// 			}
// 			return
// 		},
// 	); chk.E(err) {
// 		return
// 	}
// 	return
// }
