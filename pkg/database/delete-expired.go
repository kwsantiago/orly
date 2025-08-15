package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/pkg/database/indexes"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"time"
)

func (d *D) DeleteExpired() {
	var err error
	var expiredSerials types.Uint40s
	// make the operation atomic and save on accesses to the system clock by
	// setting the boundary at the current second
	now := time.Now().Unix()
	// search the expiration indexes for expiry timestamps that are now past
	if err = d.View(
		func(txn *badger.Txn) (err error) {
			exp, ser := indexes.ExpirationVars()
			expPrf := new(bytes.Buffer)
			if _, err = indexes.ExpirationPrefix.Write(expPrf); chk.E(err) {
				return
			}
			it := txn.NewIterator(badger.IteratorOptions{Prefix: expPrf.Bytes()})
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				key := item.KeyCopy(nil)
				buf := bytes.NewBuffer(key)
				if err = indexes.ExpirationDec(
					exp, ser,
				).UnmarshalRead(buf); chk.E(err) {
					continue
				}
				if int64(exp.Get()) > now {
					// not expired yet
					continue
				}
				expiredSerials = append(expiredSerials, ser)
			}
			return
		},
	); chk.E(err) {
	}
	// delete the events and their indexes
	for _, ser := range expiredSerials {
		var ev *event.E
		if ev, err = d.FetchEventBySerial(ser); chk.E(err) {
			continue
		}
		if err = d.DeleteEventBySerial(context.Bg(), ser, ev); chk.E(err) {
			continue
		}
	}
}
