package database

import (
	"github.com/dgraph-io/badger/v4"
	"not.realy.lol/chk"
	"not.realy.lol/codecbuf"
	"not.realy.lol/context"
	"not.realy.lol/event"
)

// SaveEvent saves an event to the database, generating all the necessary indexes.
func (d *D) SaveEvent(c context.T, ev *event.E) (err error) {
	// Get a buffer from the pool
	buf := codecbuf.Get()
	defer codecbuf.Put(buf)

	// Marshal the event to binary
	ev.MarshalBinary(buf)

	// Get the next sequence number for the event
	var serial uint64
	if serial, err = d.seq.Next(); chk.E(err) {
		return
	}

	// Generate all indexes for the event
	indexes := GenerateIndexes(ev, serial)

	// Start a transaction to save the event and all its indexes
	err = d.Update(func(txn *badger.Txn) error {
		// Save each index
		for _, idx := range indexes {
			// Get a buffer from the pool for each index
			idxBuf := codecbuf.Get()
			defer codecbuf.Put(idxBuf)

			// Marshal the index to binary
			if err := idx.MarshalWrite(idxBuf); chk.E(err) {
				return err
			}

			// Save the index to the database
			if err := txn.Set(idxBuf.Bytes(), buf.Bytes()); chk.E(err) {
				return err
			}
		}
		return nil
	})

	return
}
