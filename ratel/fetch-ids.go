package ratel

import (
	"io"
	"orly.dev/chk"
	"orly.dev/eventidserial"

	"github.com/dgraph-io/badger/v4"

	"orly.dev/context"
	"orly.dev/ratel/keys/id"
	"orly.dev/ratel/keys/serial"
	"orly.dev/ratel/prefixes"
	"orly.dev/tag"
)

// FetchIds retrieves events based on a list of event Ids that have been provided.
func (r *T) FetchIds(c context.T, evIds *tag.T, out io.Writer) (err error) {
	// create an ample buffer for decoding events, 100kb should usually be enough, if
	// it needs to get bigger it will be reallocated.
	b := make([]byte, 0, 100000)
	err = r.View(
		func(txn *badger.Txn) (err error) {
			for _, v := range evIds.ToSliceOfBytes() {
				var evId *id.T
				if evId, err = id.NewFromBytes(v); chk.E(err) {
					return
				}
				k := prefixes.Id.Key(evId)
				it := txn.NewIterator(badger.DefaultIteratorOptions)
				var ser *serial.T
				defer it.Close()
				for it.Seek(k); it.ValidForPrefix(k); it.Next() {
					key := it.Item().Key()
					ser = serial.FromKey(key)
					break
				}
				var item *badger.Item
				if item, err = txn.Get(prefixes.Event.Key(ser)); chk.E(err) {
					return
				}
				if b, err = item.ValueCopy(nil); chk.E(err) {
					return
				}
				// if r.UseCompact {
				//	ev := &event.E{}
				//	var rem []byte
				//	if rem, err = ev.UnmarshalCompact(b); chk.E(err) {
				//		return
				//	}
				//	if len(rem) > 0 {
				//		log.I.S(rem)
				//	}
				//	if _, err = out.Write(ev.Serialize()); chk.E(err) {
				//		return
				//	}
				// } else {
				// if db isn't using compact encoding the bytes are already right
				if _, err = out.Write(b); chk.E(err) {
					return
				}
				// }
				// add the new line after entries
				if _, err = out.Write([]byte{'\n'}); chk.E(err) {
					return
				}
			}
			return
		},
	)
	return
}

func (r *T) EventIdsBySerial(start uint64, count int) (
	evs []eventidserial.E, err error,
) {
	// TODO implement me
	panic("implement me")
}
