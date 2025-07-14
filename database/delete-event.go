package database

import (
	"bytes"
	"github.com/dgraph-io/badger/v4"
	"orly.dev/database/indexes"
	"orly.dev/encoders/eventid"
	"orly.dev/utils/context"
	"orly.dev/utils/log"
	"time"
)

func (d *D) DeleteEvent(
	c context.T, eid *eventid.T, noTombstone ...bool,
) (err error) {
	d.Logger.Warningf("deleting event %0x", eid.Bytes())

	// Get the serial number for the event ID
	ser, err := d.GetSerialById(eid.Bytes())
	if err != nil {
		return
	}
	if ser == nil {
		// Event not found, nothing to delete
		return
	}
	// Fetch the event to get its data
	ev, err := d.FetchEventBySerial(ser)
	if err != nil {
		return
	}
	if ev == nil {
		// Event not found, nothing to delete
		return
	}
	// Get all indexes for the event
	idxs, err := GetIndexesForEvent(ev, ser.Get())
	if err != nil {
		return
	}
	// Create a tombstone key if requested
	var tombstoneKey []byte
	if len(noTombstone) == 0 || !noTombstone[0] {
		log.I.F("making tombstone for event %0x", eid.Bytes())
		// Create a tombstone key using the Tombstone type from indexes
		fid, ts := indexes.TombstoneVars()
		// Set the event ID
		if err = fid.FromId(eid.Bytes()); err != nil {
			return
		}
		// Set the current timestamp
		ts.Set(uint64(time.Now().Unix()))
		// Create the tombstone key using the proper encoder
		buf := new(bytes.Buffer)
		if err = indexes.TombstoneEnc(fid, ts).MarshalWrite(buf); err != nil {
			return
		}
		tombstoneKey = buf.Bytes()
	}
	// Get the event key
	eventKey := new(bytes.Buffer)
	if err = indexes.EventEnc(ser).MarshalWrite(eventKey); err != nil {
		return
	}
	// Delete the event and all its indexes in a transaction
	err = d.Update(
		func(txn *badger.Txn) (err error) {
			// Delete the event
			if err = txn.Delete(eventKey.Bytes()); err != nil {
				return
			}
			// Delete all indexes
			for _, key := range idxs {
				if err = txn.Delete(key); err != nil {
					return
				}
			}
			// Write the tombstone if requested
			if len(tombstoneKey) > 0 {
				log.D.F("writing tombstone %0x", tombstoneKey)
				log.W.F(
					"writing tombstone %0x for event %0x", tombstoneKey,
					eid.Bytes(),
				)
				if err = txn.Set(tombstoneKey, nil); err != nil {
					return
				}
			}
			return
		},
	)
	return
}
