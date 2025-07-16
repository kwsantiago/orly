package socketapi

import (
	"github.com/fasthttp/websocket"
	"net/http"

	"orly.dev/protocol/ws"
)

const (
	DefaultChallengeHRP    = "nchal"
	DefaultChallengeLength = 16
)

// GetListener generates a new ws.Listener with a new challenge for a
// subscriber.
func GetListener(conn *websocket.Conn, req *http.Request) (w *ws.Listener) {
	// var err error
	// cb := make([]byte, DefaultChallengeLength)
	// if _, err = rand.Read(cb); chk.E(err) {
	// 	panic(err)
	// }
	// var b5 []byte
	// if b5, err = bech32encoding.ConvertForBech32(cb); chk.E(err) {
	// 	return
	// }
	// var encoded []byte
	// if encoded, err = bech32.Encode(
	// 	[]byte(DefaultChallengeHRP), b5,
	// ); chk.E(err) {
	// 	return
	// }
	w = ws.NewListener(conn, req)
	return
}
