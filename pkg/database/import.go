package database

import (
	"bufio"
	"io"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"os"
	"runtime/debug"
)

const maxLen = 500000000

// Import a collection of events in line structured minified JSON format (JSONL).
func (d *D) Import(rr io.Reader) {
	// store to disk so we can return fast
	tmpPath := os.TempDir() + string(os.PathSeparator) + "orly"
	os.MkdirAll(tmpPath, 0700)
	tmp, err := os.CreateTemp(tmpPath, "")
	if chk.E(err) {
		return
	}
	log.I.F("buffering upload to %s", tmp.Name())
	if _, err = io.Copy(tmp, rr); chk.E(err) {
		return
	}
	if _, err = tmp.Seek(0, 0); chk.E(err) {
		return
	}

	go func() {
		var err error
		// Create a scanner to read the buffer line by line
		scan := bufio.NewScanner(tmp)
		scanBuf := make([]byte, maxLen)
		scan.Buffer(scanBuf, maxLen)

		var count, total int
		for scan.Scan() {
			select {
			case <-d.ctx.Done():
				log.I.F("context closed")
				return
			default:
			}

			b := scan.Bytes()
			total += len(b) + 1
			if len(b) < 1 {
				continue
			}

			ev := &event.E{}
			if _, err = ev.Unmarshal(b); err != nil {
				continue
			}

			if _, _, err = d.SaveEvent(d.ctx, ev, false, nil); err != nil {
				continue
			}

			b = nil
			ev = nil
			count++
			if count%100 == 0 {
				log.I.F("received %d events", count)
				debug.FreeOSMemory()
			}
		}

		log.I.F("read %d bytes and saved %d events", total, count)
		err = scan.Err()
		if chk.E(err) {
		}

		// Help garbage collection
		tmp = nil
	}()

	return
}
