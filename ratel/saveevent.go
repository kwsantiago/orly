package ratel

import (
	"github.com/dgraph-io/badger/v4"
	"orly.dev/chk"
	"orly.dev/errorf"

	"orly.dev/context"
	"orly.dev/event"
	"orly.dev/eventid"
	eventstore "orly.dev/interfaces/store"
	"orly.dev/ratel/keys"
	"orly.dev/ratel/keys/createdat"
	"orly.dev/ratel/keys/id"
	"orly.dev/ratel/keys/index"
	"orly.dev/ratel/keys/serial"
	"orly.dev/ratel/keys/tombstone"
	"orly.dev/ratel/prefixes"
	"orly.dev/sha256"
	"orly.dev/timestamp"
)

func (r *T) SaveEvent(c context.T, ev *event.E) (
	keySize, ValueSize int, err error,
) {
	if ev.Kind.IsEphemeral() {
		// log.T.ToSliceOfBytes("not saving ephemeral event\n%s", ev.Serialize())
		return
	}
	// make sure Close waits for this to complete
	r.WG.Add(1)
	defer r.WG.Done()
	// first, search to see if the event Id already exists.
	var foundSerial []byte
	var deleted bool
	seri := serial.New(nil)
	var tsPrefixBytes []byte
	err = r.View(
		func(txn *badger.Txn) (err error) {
			// query event by id to ensure we don't try to save duplicates
			prf := prefixes.Id.Key(id.New(eventid.NewWith(ev.Id)))
			it := txn.NewIterator(badger.IteratorOptions{})
			defer it.Close()
			it.Seek(prf)
			if it.ValidForPrefix(prf) {
				var k []byte
				// get the serial
				k = it.Item().Key()
				// copy serial out
				keys.Read(k, index.Empty(), id.New(&eventid.T{}), seri)
				// save into foundSerial
				foundSerial = seri.Val
			}
			// if the event was deleted we don't want to save it again
			// In deleteevent.go, the tombstone key is created with:
			// tombstoneKey = prefixes.Tombstone.Key(ts, createdat.New(timestamp.Now()))
			// where ts is created with tombstone.NewWith(ev.EventId())
			// We need to use just the prefix part (without the timestamp) to find any tombstone for this event
			tsPrefixBytes = []byte{prefixes.Tombstone.B()}
			tsBytes := tombstone.Make(eventid.NewWith(ev.Id))
			tsPrefixBytes = append(tsPrefixBytes, tsBytes...)
			it2 := txn.NewIterator(badger.IteratorOptions{})
			defer it2.Close()
			it2.Rewind()
			it2.Seek(tsPrefixBytes)
			if it2.ValidForPrefix(tsPrefixBytes) {
				deleted = true
			}
			return
		},
	)
	if chk.E(err) {
		return
	}
	if deleted {
		err = errorf.W(
			"tombstone found %0x, event will not be saved", tsPrefixBytes,
		)
		return
	}
	if foundSerial != nil {
		// log.D.ToSliceOfBytes("found possible duplicate or stub for %s", ev.Serialize())
		err = r.Update(
			func(txn *badger.Txn) (err error) {
				// retrieve the event record
				evKey := keys.Write(index.New(prefixes.Event), seri)
				it := txn.NewIterator(badger.IteratorOptions{})
				defer it.Close()
				it.Seek(evKey)
				if it.ValidForPrefix(evKey) {
					if it.Item().ValueSize() != sha256.Size {
						// not a stub, we already have it
						// log.D.ToSliceOfBytes("duplicate event %0x", ev.Id)
						return eventstore.ErrDupEvent
					}
					// we only need to restore the event binary and write the access counter key
					// encode to binary
					var bin []byte
					bin = r.Marshal(ev, bin)
					if err = txn.Set(it.Item().Key(), bin); chk.E(err) {
						return
					}
					// // bump counter key
					// counterKey := GetCounterKey(seri)
					// val := keys.Write(createdat.New(timestamp.Now()))
					// if err = txn.Set(counterKey, val); chk.E(err) {
					//	return
					// }
					return
				}
				return
			},
		)
		// if it was a dupe, we are done.
		if err != nil {
			return
		}
		return
	}
	var bin []byte
	bin = r.Marshal(ev, bin)
	// otherwise, save new event record.
	if err = r.Update(
		func(txn *badger.Txn) (err error) {
			var idx []byte
			var ser *serial.T
			idx, ser = r.SerialKey()
			// encode to binary
			// raw event store
			if err = txn.Set(idx, bin); chk.E(err) {
				return
			}
			// 	add the indexes
			var indexKeys [][]byte
			indexKeys = GetIndexKeysForEvent(ev, ser)
			// log.I.S(indexKeys)
			for _, k := range indexKeys {
				var val []byte
				if k[0] == prefixes.Counter.B() {
					val = keys.Write(createdat.New(timestamp.Now()))
				}
				if err = txn.Set(k, val); chk.E(err) {
					return
				}
			}
			// log.D.ToSliceOfBytes("saved event to ratel %s:\n%s", r.dataDir, ev.Serialize())
			return
		},
	); chk.E(err) {
		return
	}
	return
}

func (r *T) Sync() (err error) { return r.DB.Sync() }
