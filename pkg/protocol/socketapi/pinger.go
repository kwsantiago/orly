package socketapi

import (
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"time"

	"github.com/fasthttp/websocket"
)

// Pinger sends periodic WebSocket ping messages to ensure the connection is
// alive and responsive. It terminates the connection if pings fail or the
// context is canceled.
//
// Parameters:
//
//   - ctx: A context object used to monitor cancellation signals and
//     manage termination of the method execution.
//
//   - ticker: A time.Ticker object that triggers periodic pings based on
//     its configured interval.
//
//   - cancel: A context.CancelFunc called to gracefully terminate operations
//     associated with the WebSocket connection.
//
//   - s: An interface representing the server context, allowing interactions
//     related to the connection.
//
// Expected behavior:
//
// The method writes ping messages to the WebSocket connection at intervals
// dictated by the ticker. If the ping write fails or the context is canceled,
// it stops the ticker, invokes the cancel function, and closes the connection.
func (a *A) Pinger(
	ctx context.T, ticker *time.Ticker, cancel context.F, s server.I,
) {
	defer func() {
		cancel()
		ticker.Stop()
		_ = a.Listener.Conn.Close()
	}()
	var err error
	for {
		select {
		case <-ticker.C:
			err = a.Listener.Conn.WriteControl(
				websocket.PingMessage, nil,
				time.Now().Add(DefaultPingWait),
			)
			if err != nil {
				log.E.F("error writing ping: %v; closing websocket", err)
				return
			}
			a.Listener.RealRemote()
		case <-ctx.Done():
			return
		}
	}
}
