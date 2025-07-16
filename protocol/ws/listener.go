// Package ws implements nostr websockets with their authentication state.
package ws

import (
	"net/http"
	"strings"
	"sync"

	"github.com/fasthttp/websocket"

	"orly.dev/app/realy/helpers"
	"orly.dev/utils/atomic"
)

// Listener is a websocket implementation for a relay listener.
type Listener struct {
	mutex   sync.Mutex
	Conn    *websocket.Conn
	Request *http.Request
	remote  atomic.String
}

// NewListener creates a new Listener for listening for inbound connections for
// a relay.
func NewListener(conn *websocket.Conn, req *http.Request) (ws *Listener) {
	ws = &Listener{Conn: conn, Request: req}
	ws.setRemoteFromReq(req)
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
