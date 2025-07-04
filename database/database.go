package database

import (
	"github.com/dgraph-io/badger/v4"
	"io"
	"not.realy.lol/chk"
	"not.realy.lol/context"
	"not.realy.lol/event"
	"not.realy.lol/eventid"
	"not.realy.lol/eventidserial"
	"not.realy.lol/filter"
	"not.realy.lol/interfaces/store"
	"not.realy.lol/log"
	"not.realy.lol/lol"
	"not.realy.lol/units"
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
	opts := badger.DefaultOptions(d.dataDir)
	opts.BlockCacheSize = int64(units.Gb)
	opts.BlockSize = units.Gb
	opts.CompactL0OnClose = true
	opts.LmaxCompaction = true
	if d.DB, err = badger.Open(opts); chk.E(err) {
		return
	}
	log.I.Ln("getting event sequence lease", d.dataDir)
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

func (d *D) QueryEvents(c context.T, f *filter.T) (evs event.S, err error) {
	// TODO implement me
	panic("implement me")
}

func (d *D) QueryForIds(c context.T, f *filter.T) (
	evs []store.IdTsPk, err error,
) {
	// TODO implement me
	panic("implement me")
}

func (d *D) DeleteEvent(
	c context.T, ev *eventid.T, noTombstone ...bool,
) (err error) {
	// TODO implement me
	panic("implement me")
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
