package socketapi

import (
	"github.com/fasthttp/websocket"
	"net/http"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/helpers"
	"orly.dev/interfaces/server"
	"orly.dev/log"
	"orly.dev/publish"
	"orly.dev/servemux"
	"orly.dev/units"
	"orly.dev/ws"
	"strings"
	"time"
)

type SocketParams struct {
	WriteWait      time.Duration
	PongWait       time.Duration
	PingWait       time.Duration
	MaxMessageSize int64
}

func DefaultSocketParams() *SocketParams {
	return &SocketParams{
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingWait:       30 * time.Second,
		MaxMessageSize: 1 * units.Mb,
	}
}

type A struct {
	Ctx context.T
	server.I
	// Web is an optional web server that appears on `/` with no Upgrade for
	// websockets or Accept for application/nostr+json present.
	Web http.Handler
	*SocketParams
	Listener *ws.Listener
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize: 1024, WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func New(s server.I, path string, sm *servemux.S, socketParams *SocketParams) {
	a := &A{I: s, SocketParams: socketParams}
	sm.Handle(path, a)
	return
}

// ServeHTTP handles incoming HTTP requests and processes them accordingly. It
// serves the relayinfo for specific headers or delegates to a web handler. It
// processes WebSocket upgrade requests when applicable.
func (a *A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote := helpers.GetRemoteFromReq(r)
	if r.Header.Get("Upgrade") != "websocket" &&
		r.Header.Get("Accept") == "application/nostr+json" {
		log.T.F("serving relayinfo %s", remote)
		a.I.HandleRelayInfo(w, r)
		return
	}
	if r.Header.Get("Upgrade") != "websocket" {
		if a.Web == nil {
			a.I.HandleRelayInfo(w, r)
		} else {
			a.Web.ServeHTTP(w, r)
		}
		return
	}
	var err error
	ticker := time.NewTicker(a.PingWait)
	var cancel context.F
	a.Ctx, cancel = context.Cancel(a.I.Context())
	var conn *websocket.Conn
	if conn, err = Upgrader.Upgrade(w, r, nil); err != nil {
		log.E.F("%s failed to upgrade websocket: %v", remote, err)
		return
	}
	log.T.F(
		"upgraded to websocket %s (remote %s local %s)", remote,
		conn.RemoteAddr(), conn.LocalAddr(),
	)
	a.Listener = ws.NewListener(conn, r)
	defer func() {
		log.T.F("remote %s closed connection", remote)
		cancel()
		ticker.Stop()
		publish.P.Receive(
			&W{
				Cancel: true,
				I:      a.Listener,
			},
		)
		chk.E(a.Listener.Close())
	}()
	conn.SetReadLimit(a.MaxMessageSize)
	chk.E(conn.SetReadDeadline(time.Now().Add(a.PongWait)))
	conn.SetPongHandler(
		func(string) error {
			chk.E(conn.SetReadDeadline(time.Now().Add(a.PongWait)))
			return nil
		},
	)
	go a.Pinger(a.Ctx, ticker, cancel, remote)
	var message []byte
	var typ int
	for {
		select {
		case <-a.Ctx.Done():
			log.I.F("%s closing connection", remote)
			a.Listener.Close()
			return
		default:
		}
		typ, message, err = conn.ReadMessage()
		if err != nil {
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
					a.Listener.Request.Header.Get("X-Forwarded-For"), err,
				)
			}
			return
		}
		if typ == websocket.PingMessage {
			log.T.F("pinging %s", remote)
			if _, err = a.Listener.Write(nil); chk.E(err) {
			}
			continue
		}
		go a.HandleMessage(message, remote)
	}

}

func (a *A) Pinger(
	ctx context.T, ticker *time.Ticker, cancel context.F, remote string,
) {
	log.T.F("running pinger for %s", remote)
	defer func() {
		cancel()
		ticker.Stop()
		_ = a.Listener.Conn.Close()
		log.T.F("stopped pinger for %s", remote)
	}()
	var err error
	for {
		select {
		case <-ticker.C:
			err = a.Listener.Conn.WriteControl(
				websocket.PingMessage, nil,
				time.Now().Add(a.PingWait),
			)
			if err != nil {
				log.E.F(
					"%s error writing ping: %v; closing websocket", remote, err,
				)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
