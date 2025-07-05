// Package ws implements nostr websockets with their authentication state.
package ws

import (
	"net/http"
	"orly.dev/helpers"
	"strings"
	"sync"

	"github.com/fasthttp/websocket"

	"go.uber.org/atomic"
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
func NewListener(
	conn *websocket.Conn,
	req *http.Request,
) (ws *Listener) {
	ws = &Listener{Conn: conn, Request: req}
	ws.remote.Store(helpers.GetRemoteFromReq(req))
	return
}

// Write a message to send to a client.
func (ws *Listener) Write(p []byte) (n int, err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	err = ws.Conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		n = len(p)
		if strings.Contains(err.Error(), "close sent") {
			_ = ws.Close()
			err = nil
			return
		}
	}
	return
}

// Remote returns the stored remote address of the client.
func (ws *Listener) Remote() string {
	return ws.remote.Load()
}

// Req returns the http.Request associated with the client connection to the
// Listener.
func (ws *Listener) Req() *http.Request {
	return ws.Request
}

// Close the Listener connection from the Listener side.
func (ws *Listener) Close() (err error) { return ws.Conn.Close() }
