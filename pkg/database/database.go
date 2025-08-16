package database

import (
	"github.com/dgraph-io/badger/v4"
	"orly.dev/pkg/encoders/eventidserial"
	"orly.dev/pkg/utils/apputil"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
	"orly.dev/pkg/utils/units"
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
	opts.Logger = d.Logger
	if d.DB, err = badger.Open(opts); chk.E(err) {
		return
	}
	log.T.Ln("getting event sequence lease", d.dataDir)
	if d.seq, err = d.DB.GetSequence([]byte("EVENTS"), 1000); chk.E(err) {
		return
	}
	// run code that updates indexes when new indexes have been added and bumps
	// the version so they aren't run again.
	d.RunMigrations()
	// start up the expiration tag processing and shut down and clean up the
	// database after the context is canceled.
	go func() {
		expirationTicker := time.NewTicker(time.Minute * 10)
		select {
		case <-expirationTicker.C:
			d.DeleteExpired()
			return
		case <-d.ctx.Done():
		}
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
	d.DB.RunValueLogGC(0.5)
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
