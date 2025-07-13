package database

import (
	"bytes"
	"encoding/binary"
	"github.com/dgraph-io/badger/v4"
	"io"
	"orly.dev/database/indexes"
	"orly.dev/encoders/eventid"
	"orly.dev/encoders/eventidserial"
	"orly.dev/utils/apputil"
	"orly.dev/utils/chk"
	"orly.dev/utils/context"
	"orly.dev/utils/log"
	"orly.dev/utils/lol"
	"orly.dev/utils/units"
	"os"
	"path/filepath"
	"time"
)

type D struct {
	ctx     context.T
	cancel  context.F
	dataDir string
	Logger  *logger
	*badger.DB
	seq *badger.Sequence
}

func New(ctx context.T, cancel context.F, dataDir, logLevel string) (
	d *D, err error,
) {
	d = &D{
		ctx:     ctx,
		cancel:  cancel,
		dataDir: dataDir,
		Logger:  NewLogger(lol.GetLogLevel(logLevel), dataDir),
		DB:      nil,
		seq:     nil,
	}

	// Ensure the data directory exists
	if err = os.MkdirAll(dataDir, 0755); chk.E(err) {
		return
	}

	// Also ensure the directory exists using apputil.EnsureDir for any potential subdirectories
	dummyFile := filepath.Join(dataDir, "dummy.sst")
	if err = apputil.EnsureDir(dummyFile); chk.E(err) {
		return
	}

	opts := badger.DefaultOptions(d.dataDir)
	opts.BlockCacheSize = int64(units.Gb)
	opts.BlockSize = units.Gb
	opts.CompactL0OnClose = true
	opts.LmaxCompaction = true
	if d.DB, err = badger.Open(opts); chk.E(err) {
		return
	}
	log.T.Ln("getting event sequence lease", d.dataDir)
	if d.seq, err = d.DB.GetSequence([]byte("EVENTS"), 1000); chk.E(err) {
		return
	}
	go func() {
		<-d.ctx.Done()
		d.cancel()
		d.seq.Release()
		d.DB.Close()
	}()
	return
}

// Path returns the path where the database files are stored.
func (d *D) Path() string { return d.dataDir }

func (d *D) Wipe() (err error) {
	// TODO implement me
	panic("implement me")
}

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
		// Create a tombstone key using the event ID and current timestamp
		// Since we don't have a dedicated tombstone prefix in the database package,
		// we'll use a custom prefix "tmb" for tombstones
		buf := new(bytes.Buffer)
		// Write the tombstone prefix
		buf.Write([]byte("tmb"))
		// Write the event ID
		buf.Write(eid.Bytes())
		// Write the current timestamp
		ts := uint64(time.Now().Unix())
		binary.BigEndian.PutUint64(make([]byte, 8), ts)
		buf.Write(make([]byte, 8))
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

func (d *D) Import(r io.Reader) {
	// TODO implement me
	panic("implement me")
}

func (d *D) Export(c context.T, w io.Writer, pubkeys ...[]byte) {
	// TODO implement me
	panic("implement me")
}

func (d *D) SetLogLevel(level string) {
	d.Logger.SetLogLevel(lol.GetLogLevel(level))
}

func (d *D) EventIdsBySerial(start uint64, count int) (
	evs []eventidserial.E, err error,
) {
	// TODO implement me
	panic("implement me")
}

// Init initializes the database with the given path.
func (d *D) Init(path string) (err error) {
	// The database is already initialized in the New function,
	// so we just need to ensure the path is set correctly.
	d.dataDir = path
	return nil
}

// Sync flushes the database buffers to disk.
func (d *D) Sync() (err error) {
	return d.DB.Sync()
}

// Close releases resources and closes the database.
func (d *D) Close() (err error) {
	if d.seq != nil {
		if err = d.seq.Release(); chk.E(err) {
			return
		}
	}
	if d.DB != nil {
		if err = d.DB.Close(); chk.E(err) {
			return
		}
	}
	return
}
