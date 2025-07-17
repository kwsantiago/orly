package socketapi

import (
	"net/http"

	"github.com/fasthttp/websocket"
)

// Upgrader is a preconfigured instance of websocket.Upgrader used to upgrade
// HTTP connections to WebSocket connections with specific buffer sizes and a
// permissive origin-checking function.
var Upgrader = websocket.Upgrader{
	ReadBufferSize: 1024, WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
