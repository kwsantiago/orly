package socketapi

import (
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/envelopes/authenvelope"
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/units"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
)

const (
	DefaultWriteWait      = 10 * time.Second
	DefaultPongWait       = 60 * time.Second
	DefaultPingWait       = DefaultPongWait / 2
	DefaultMaxMessageSize = 1 * units.Mb
)

// A is a composite type that integrates a context, a websocket Listener, and a
// server interface to manage WebSocket-based server communication. It is
// designed to handle message processing, authentication, and event dispatching
// in its operations.
type A struct {
	Ctx context.T
	*ws.Listener
	server.I
}

// Serve handles an incoming WebSocket request by upgrading the HTTP request,
// managing the WebSocket connection, and delegating received messages for
// processing.
//
// # Parameters
//
//   - w: The HTTP response writer used to manage the connection upgrade.
//
//   - r: The HTTP request object that is being upgraded to a WebSocket
//     connection.
//
//   - s: The server context object that manages request lifecycle and state.
//
// Expected behavior:
//
// The method upgrades the HTTP connection to a WebSocket connection, sets up
// read and write limits, handles pings and pongs for keeping the connection
// alive, and processes incoming messages. It ensures proper cleanup of
// resources on connection termination or cancellation, adhering to the given
// context's lifecycle.
func (a *A) Serve(w http.ResponseWriter, r *http.Request, s server.I) {
	var err error
	ticker := time.NewTicker(DefaultPingWait)
	var cancel context.F
	a.Ctx, cancel = context.Cancel(s.Context())
	var conn *websocket.Conn
	conn, err = Upgrader.Upgrade(w, r, nil)
	if chk.E(err) {
		log.E.F("failed to upgrade websocket: %v", err)
		return
	}
	a.Listener = ws.NewListener(conn, r, a.I.AuthRequired())
	defer func() {
		cancel()
		ticker.Stop()
		a.Publisher().Receive(
			&W{
				Cancel:   true,
				Listener: a.Listener,
			},
		)
		chk.E(a.Listener.Conn.Close())
	}()
	conn.SetReadLimit(DefaultMaxMessageSize)
	chk.E(conn.SetReadDeadline(time.Now().Add(DefaultPongWait)))
	conn.SetPongHandler(
		func(string) error {
			chk.E(conn.SetReadDeadline(time.Now().Add(DefaultPongWait)))
			return nil
		},
	)
	if a.I.AuthRequired() {
		log.T.F("requesting auth from client from %s", a.Listener.RealRemote())
		a.Listener.RequestAuth()
		if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).
			Write(a.Listener); chk.E(err) {
			return
		}
	}
	go a.Pinger(a.Ctx, ticker, cancel, a.I)
	var message []byte
	var typ int
	for {
		select {
		case <-a.Ctx.Done():
			a.Listener.Close()
			return
		case <-s.Context().Done():
			a.Listener.Close()
			return
		default:
		}
		if typ, message, err = conn.ReadMessage(); err != nil {
			if strings.Contains(
				err.Error(), "use of closed network connection",
			) {
				return
			}
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
				websocket.CloseAbnormalClosure,
			) {
				log.W.F(
					"unexpected close error from %s: %v",
					helpers.GetRemoteFromReq(r), err,
				)
			}
			return
		}
		if typ == websocket.PingMessage {
			if err = a.Listener.WriteMessage(
				websocket.PongMessage, nil,
			); chk.E(err) {
			}
			continue
		}
		go a.HandleMessage(message)
	}
}
