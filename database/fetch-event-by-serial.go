package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/encoders/codecbuf"
	"orly.dev/encoders/event"
	"orly.dev/utils/chk"
)

func (d *D) FetchEventBySerial(ser *types.Uint40) (ev *event.E, err error) {
	if err = d.View(
		func(txn *badger.Txn) (err error) {
			buf := codecbuf.Get()
			defer codecbuf.Put(buf)
			if err = indexes.EventEnc(ser).MarshalWrite(buf); chk.E(err) {
				return
			}
			var item *badger.Item
			if item, err = txn.Get(buf.Bytes()); chk.E(err) {
				return
			}
			var v []byte
			if v, err = item.ValueCopy(nil); chk.E(err) {
				return
			}
			ev = new(event.E)
			if err = ev.UnmarshalBinary(bytes.NewBuffer(v)); chk.E(err) {
				return
			}
			return
		},
	); err != nil {
		return
	}
	return
}
