package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/pkg/database/indexes"
	"orly.dev/pkg/database/indexes/types"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/eventid"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
)

// DeleteEvent removes an event from the database identified by `eid`. If
// noTombstone is false or not provided, a tombstone is created for the event.
func (d *D) DeleteEvent(c context.T, eid *eventid.T) (err error) {
	d.Logger.Warningf("deleting event %0x", eid.Bytes())

	// Get the serial number for the event ID
	var ser *types.Uint40
	ser, err = d.GetSerialById(eid.Bytes())
	if chk.E(err) {
		return
	}
	if ser == nil {
		// Event not found, nothing to delete
		return
	}
	// Fetch the event to get its data
	var ev *event.E
	ev, err = d.FetchEventBySerial(ser)
	if chk.E(err) {
		return
	}
	if ev == nil {
		// Event not found, nothing to delete
		return
	}
	// Get all indexes for the event
	var idxs [][]byte
	idxs, err = GetIndexesForEvent(ev, ser.Get())
	if chk.E(err) {
		return
	}
	// Get the event key
	eventKey := new(bytes.Buffer)
	if err = indexes.EventEnc(ser).MarshalWrite(eventKey); chk.E(err) {
		return
	}
	// Delete the event and all its indexes in a transaction
	err = d.Update(
		func(txn *badger.Txn) (err error) {
			// Delete the event
			if err = txn.Delete(eventKey.Bytes()); chk.E(err) {
				return
			}
			// Delete all indexes
			for _, key := range idxs {
				if err = txn.Delete(key); chk.E(err) {
					return
				}
			}
			return
		},
	)
	return
}
