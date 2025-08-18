package ws

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/errorf"
	"orly.dev/pkg/utils/units"
	"time"

	ws "github.com/coder/websocket"
)

// Connection represents a websocket connection to a Nostr relay.
type Connection struct {
	conn *ws.Conn
}

// NewConnection creates a new websocket connection to a Nostr relay.
func NewConnection(
	ctx context.T, url string, reqHeader http.Header,
	tlsConfig *tls.Config,
) (c *Connection, err error) {
	var conn *ws.Conn
	if conn, _, err = ws.Dial(
		ctx, url, getConnectionOptions(reqHeader, tlsConfig),
	); err != nil {
		return
	}
	conn.SetReadLimit(33 * units.Mb)
	return &Connection{
		conn: conn,
	}, nil
}

// WriteMessage writes arbitrary bytes to the websocket connection.
func (c *Connection) WriteMessage(
	ctx context.T, data []byte,
) (err error) {
	if err = c.conn.Write(ctx, ws.MessageText, data); err != nil {
		err = errorf.E("failed to write message: %w", err)
		return
	}
	return nil
}

// ReadMessage reads arbitrary bytes from the websocket connection into the provided buffer.
func (c *Connection) ReadMessage(
	ctx context.T, buf io.Writer,
) (err error) {
	var reader io.Reader
	if _, reader, err = c.conn.Reader(ctx); err != nil {
		err = errorf.E("failed to get reader: %w", err)
		return
	}
	if _, err = io.Copy(buf, reader); err != nil {
		err = errorf.E("failed to read message: %w", err)
		return
	}
	return
}

// Close closes the websocket connection.
func (c *Connection) Close() error {
	return c.conn.Close(ws.StatusNormalClosure, "")
}

// Ping sends a ping message to the websocket connection.
func (c *Connection) Ping(ctx context.T) error {
	ctx, cancel := context.TimeoutCause(
		ctx, time.Millisecond*800, errors.New("ping took too long"),
	)
	defer cancel()
	return c.conn.Ping(ctx)
}
