package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"io"
	"orly.dev/pkg/database/indexes"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/codecbuf"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/units"
)

// Export the complete database of stored events to an io.Writer in line structured minified
// JSON.
func (d *D) Export(c context.T, w io.Writer, pubkeys ...[]byte) {
	var err error
	if len(pubkeys) == 0 {
		if err = d.View(
			func(txn *badger.Txn) (err error) {
				buf := codecbuf.Get()
				defer codecbuf.Put(buf)
				if err = indexes.EventEnc(nil).MarshalWrite(buf); chk.E(err) {
					return
				}
				it := txn.NewIterator(badger.IteratorOptions{Prefix: buf.Bytes()})
				evB := make([]byte, 0, units.Mb)
				defer it.Close()
				for it.Rewind(); it.Valid(); it.Next() {
					item := it.Item()
					if evB, err = item.ValueCopy(evB); chk.E(err) {
						continue
					}
					evBuf := bytes.NewBuffer(evB)
					ev := event.New()
					if err = ev.UnmarshalBinary(evBuf); chk.E(err) {
						continue
					}
					// Serialize the event to JSON and write it to the output
					if _, err = w.Write(ev.Serialize()); chk.E(err) {
						continue
					}
					if _, err = w.Write([]byte{'\n'}); chk.E(err) {
						continue
					}
				}
				return
			},
		); err != nil {
			return
		}
	} else {
		for _, pubkey := range pubkeys {
			if err = d.View(
				func(txn *badger.Txn) (err error) {
					pkBuf := codecbuf.Get()
					defer codecbuf.Put(pkBuf)
					ph := &types.PubHash{}
					if err = ph.FromPubkey(pubkey); chk.E(err) {
						return
					}
					if err = indexes.PubkeyEnc(
						ph, nil, nil,
					).MarshalWrite(pkBuf); chk.E(err) {
						return
					}
					it := txn.NewIterator(badger.IteratorOptions{Prefix: pkBuf.Bytes()})
					evB := make([]byte, 0, units.Mb)
					defer it.Close()
					for it.Rewind(); it.Valid(); it.Next() {
						item := it.Item()
						if evB, err = item.ValueCopy(evB); chk.E(err) {
							continue
						}
						evBuf := bytes.NewBuffer(evB)
						ev := event.New()
						if err = ev.UnmarshalBinary(evBuf); chk.E(err) {
							continue
						}
						// Serialize the event to JSON and write it to the output
						if _, err = w.Write(ev.Serialize()); chk.E(err) {
							continue
						}
						if _, err = w.Write([]byte{'\n'}); chk.E(err) {
							continue
						}
					}
					return
				},
			); err != nil {
				return
			}
		}
	}
	return
}
