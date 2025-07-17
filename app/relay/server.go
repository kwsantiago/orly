package relay

import (
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"orly.dev/app/config"
	"orly.dev/app/relay/helpers"
	"orly.dev/app/relay/options"
	"orly.dev/app/relay/publish"
	"orly.dev/interfaces/relay"
	"orly.dev/protocol/servemux"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
	"strconv"
	"time"

	"github.com/rs/cors"

	"orly.dev/protocol/socketapi"
	"orly.dev/utils/context"
)

type Server struct {
	Ctx        context.T
	Cancel     context.F
	options    *options.T
	relay      relay.I
	Addr       string
	mux        *servemux.S
	httpServer *http.Server
	listeners  *publish.S
	*config.C
}

type ServerParams struct {
	Ctx      context.T
	Cancel   context.F
	Rl       relay.I
	DbPath   string
	MaxLimit int
	*config.C
}

func NewServer(sp *ServerParams, opts ...options.O) (s *Server, err error) {
	op := options.Default()
	for _, opt := range opts {
		opt(op)
	}
	if storage := sp.Rl.Storage(); storage != nil {
		if err = storage.Init(sp.DbPath); chk.T(err) {
			return nil, fmt.Errorf("storage init: %w", err)
		}
	}
	serveMux := servemux.NewServeMux()
	s = &Server{
		Ctx:       sp.Ctx,
		Cancel:    sp.Cancel,
		relay:     sp.Rl,
		mux:       serveMux,
		options:   op,
		listeners: publish.New(socketapi.New()),
		C:         sp.C,
	}
	go func() {
		if err := s.relay.Init(); chk.E(err) {
			s.Shutdown()
		}
	}()
	return s, nil
}

// ServeHTTP implements the relay's http handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// standard nostr protocol only governs the "root" path of the relay and
	// websockets
	if r.URL.Path == "/" {
		if r.Header.Get("Upgrade") == "websocket" {
			s.handleWebsocket(w, r)
			return
		}
		if r.Header.Get("Accept") == "application/nostr+json" {
			s.handleRelayInfo(w, r)
			return
		}
	}
	log.I.F(
		"http request: %s from %s",
		r.URL.String(), helpers.GetRemoteFromReq(r),
	)
	s.mux.ServeHTTP(w, r)
}

// Start up the relay.
func (s *Server) Start(
	host string, port int, started ...chan bool,
) (err error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	log.I.F("starting relay listener at %s", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.httpServer = &http.Server{
		Handler:           cors.Default().Handler(s),
		Addr:              addr,
		ReadHeaderTimeout: 7 * time.Second,
		IdleTimeout:       28 * time.Second,
	}
	for _, startedC := range started {
		close(startedC)
	}
	if err = s.httpServer.Serve(ln); errors.Is(err, http.ErrServerClosed) {
	} else if err != nil {
	}
	return nil
}

// Shutdown the relay.
func (s *Server) Shutdown() {
	log.I.Ln("shutting down relay")
	s.Cancel()
	log.W.Ln("closing event store")
	chk.E(s.relay.Storage().Close())
	log.W.Ln("shutting down relay listener")
	chk.E(s.httpServer.Shutdown(s.Ctx))
	if f, ok := s.relay.(relay.ShutdownAware); ok {
		f.OnShutdown(s.Ctx)
	}
}

// Router returns the servemux that handles paths on the HTTP server of the
// relay.
func (s *Server) Router() (router *http.ServeMux) {
	return s.mux.ServeMux
}
