package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/database/indexes"
	"orly.dev/database/indexes/types"
	"orly.dev/event"
	"orly.dev/filter"
	"orly.dev/hex"
)

// SaveEvent saves an event to the database, generating all the necessary indexes.
func (d *D) SaveEvent(c context.T, ev *event.E) (kc, vc int, err error) {
	// Get a buffer from the pool
	buf := new(bytes.Buffer)
	// Marshal the event to binary
	ev.MarshalBinary(buf)

	// Check if the event is replaceable
	if ev.Kind != nil && ev.Kind.IsReplaceable() {
		// Create a filter to find previous events of the same kind from the same pubkey
		f := filter.New()
		f.Kinds.K = append(f.Kinds.K, ev.Kind)
		f.Authors = f.Authors.Append(ev.Pubkey)

		// Query for previous events
		var prevEvents event.S
		if prevEvents, err = d.QueryEvents(c, f); chk.E(err) {
			return
		}

		// If there are previous events, log that we're replacing one
		if len(prevEvents) > 0 {
			d.Logger.Infof(
				"Saving new version of replaceable event kind %d from pubkey %s",
				ev.Kind.K, hex.Enc(ev.Pubkey),
			)
		}
	}

	// Get the next sequence number for the event
	var serial uint64
	if serial, err = d.seq.Next(); chk.E(err) {
		return
	}
	// Generate all indexes for the event
	var idxs [][]byte
	if idxs, err = GetIndexesForEvent(ev, serial); chk.E(err) {
		return
	}
	// log.I.S(idxs)
	for _, k := range idxs {
		kc += len(k)
	}
	// Start a transaction to save the event and all its indexes
	err = d.Update(
		func(txn *badger.Txn) (err error) {
			// Save each index
			for _, key := range idxs {
				if err = func() (err error) {
					// Save the index to the database
					if err = txn.Set(key, nil); chk.E(err) {
						return err
					}
					return
				}(); chk.E(err) {
					return
				}
			}
			// write the event
			k := new(bytes.Buffer)
			ser := new(types.Uint40)
			if err = ser.Set(serial); chk.E(err) {
				return
			}
			if err = indexes.EventEnc(ser).MarshalWrite(k); chk.E(err) {
				return
			}
			v := new(bytes.Buffer)
			ev.MarshalBinary(v)
			kb, vb := k.Bytes(), v.Bytes()
			kc += len(kb)
			vc += len(vb)
			// log.I.S(kb, vb)
			if err = txn.Set(kb, vb); chk.E(err) {
				return
			}
			return
		},
	)
	// log.F.F("total data written: %d bytes keys %d bytes values", kc, vc)
	return
}
