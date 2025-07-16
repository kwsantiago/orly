package app

import (
	"orly.dev/utils/log"
	"os"
	"runtime"
	"time"

	"orly.dev/utils/context"
)

func MonitorResources(c context.T) {
	tick := time.NewTicker(time.Minute * 15)
	log.I.Ln("running process", os.Args[0], os.Getpid())
	for {
		select {
		case <-c.Done():
			log.D.Ln("shutting down resource monitor")
			return
		case <-tick.C:
			log.D.Ln(
				"# goroutines", runtime.NumGoroutine(),
				"# cgo calls", runtime.NumCgoCall(),
			)
		}
	}
}
