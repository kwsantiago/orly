// Package ws implements nostr websockets with their authentication state.
package ws

import (
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/protocol/auth"
	atomic2 "orly.dev/pkg/utils/atomic"
	"strings"
	"sync"

	"github.com/fasthttp/websocket"
)

// Listener is a websocket implementation for a relay listener.
type Listener struct {
	mutex         sync.Mutex
	Conn          *websocket.Conn
	Request       *http.Request
	remote        atomic2.String
	authedPubkey  atomic2.Bytes
	authRequested atomic2.Bool
	isAuthed      atomic2.Bool
	challenge     atomic2.Bytes
	pendingEvent  *event.E
}

// NewListener creates a new Listener for listening for inbound connections for
// a relay.
func NewListener(
	conn *websocket.Conn, req *http.Request, authRequired bool,
) (ws *Listener) {
	ws = &Listener{Conn: conn, Request: req}
	ws.setRemoteFromReq(req)
	if authRequired {
		ws.SetChallenge(auth.GenerateChallenge())
		ws.SetAuthedPubkey(nil)
	}
	return
}

func (ws *Listener) setRemoteFromReq(r *http.Request) {
	// Use the helper function to get the remote address
	rr := helpers.GetRemoteFromReq(r)

	// If the helper function couldn't determine the remote address, fall back
	// to the connection's remote address
	if rr == "" {
		// if that fails, fall back to the remote (probably the proxy, unless
		// the relay is actually directly listening)
		rr = ws.Conn.NetConn().RemoteAddr().String()
	}
	ws.remote.Store(rr)
}

// Write a message to send to a client.
func (ws *Listener) Write(p []byte) (n int, err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	err = ws.Conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		n = len(p)
		if strings.Contains(err.Error(), "close sent") {
			ws.Close()
			err = nil
			return
		}
	}
	return
}

// WriteJSON encodes whatever into JSON and sends it to the client.
func (ws *Listener) WriteJSON(any interface{}) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.Conn.WriteJSON(any)
}

// WriteMessage is a wrapper around the websocket WriteMessage, which includes a
// websocket message type identifier.
func (ws *Listener) WriteMessage(t int, b []byte) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.Conn.WriteMessage(t, b)
}

// RealRemote returns the stored remote address of the client.
func (ws *Listener) RealRemote() string { return ws.remote.Load() }

// Req returns the http.Request associated with the client connection to the
// Listener.
func (ws *Listener) Req() *http.Request { return ws.Request }

// Close the Listener connection from the Listener side.
func (ws *Listener) Close() (err error) { return ws.Conn.Close() }

func (ws *Listener) IsAuthed() bool       { return ws.isAuthed.Load() }
func (ws *Listener) SetAuthed(b bool)     { ws.isAuthed.Store(b) }
func (ws *Listener) AuthedPubkey() []byte { return ws.authedPubkey.Load() }
func (ws *Listener) SetAuthedPubkey(b []byte) {
	ws.isAuthed.Store(true)
	ws.authedPubkey.Store(b)
}

func (ws *Listener) Challenge() []byte { return ws.challenge.Load() }
func (ws *Listener) SetChallenge(b []byte) {
	ws.challenge.Store(b)
}

// AuthRequested returns whether the Listener has asked for auth from the
// client.
func (ws *Listener) AuthRequested() (read bool) {
	return ws.authRequested.Load()
}

// RequestAuth stores when auth has been required from a client.
func (ws *Listener) RequestAuth() {
	ws.authRequested.Store(true)
}

func (ws *Listener) SetPendingEvent(ev *event.E) {
	ws.pendingEvent = ev
}

func (ws *Listener) GetPendingEvent() (ev *event.E) {
	ev = ws.pendingEvent
	ws.pendingEvent = nil
	return
}
