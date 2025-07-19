package app

import (
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"os"
	"runtime"
	"time"
)

// MonitorResources periodically logs resource usage metrics such as the number
// of active goroutines and CGO calls at 15-minute intervals, and exits when the
// provided context signals cancellation.
//
// # Parameters
//
//   - c: Context used to control the lifecycle of the resource monitoring process.
//
// # Expected behaviour
//
// The function runs indefinitely, logging metrics every 15 minutes until the
// context is cancelled. Upon cancellation, it logs a shutdown message and exits
// gracefully without returning any values.
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
