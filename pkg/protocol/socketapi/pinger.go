package socketapi

import (
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"time"

	"github.com/fasthttp/websocket"
)

// Pinger sends periodic WebSocket ping messages to maintain an active
// connection and handles clean-up when the context is cancelled or the ticker
// triggers.
//
// # Parameters
//
//   - ctx (context.T): The context controlling the operation lifecycle, used to
//     detect cancellation or completion signals.
//
//   - ticker (*time.Ticker): A timer that triggers periodic ping messages at
//     regular intervals.
//
//   - cancel (context.F): A function to cancel the context when the operation
//     should terminate, typically called on shutdown or error.
//
//   - s (server.I): The server interface provides contextual information for the
//     connection, though not directly used within this method.
//
// # Expected behaviour
//
// Sends WebSocket ping messages at intervals defined by the ticker. If an error
// occurs during transmission, logs the failure and closes the underlying
// connection. Cleans up resources by stopping the ticker and cancelling the
// context when the operation completes or is interrupted.
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
