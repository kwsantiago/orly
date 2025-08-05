package relay

import (
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"orly.dev/pkg/protocol/openapi"
	"orly.dev/pkg/protocol/socketapi"
	"strconv"
	"strings"
	"time"

	"orly.dev/pkg/app/config"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/app/relay/options"
	"orly.dev/pkg/app/relay/publish"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/protocol/servemux"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/keys"
	"orly.dev/pkg/utils/log"

	"github.com/rs/cors"
)

// Server represents the core structure for running a nostr relay. It
// encapsulates various components such as context, cancel function, options,
// relay interface, address, HTTP server, and configuration settings.
type Server struct {
	Ctx              context.T
	Cancel           context.F
	options          *options.T
	relay            relay.I
	Addr             string
	mux              *servemux.S
	httpServer       *http.Server
	listeners        *publish.S
	blacklistPubkeys [][]byte
	*config.C
	*Lists
	*Peers
	Mux *servemux.S
}

// ServerParams represents the configuration parameters for initializing a
// server. It encapsulates various components such as context, cancel function,
// relay interface, database path, maximum limit, and configuration settings.
type ServerParams struct {
	Ctx      context.T
	Cancel   context.F
	Rl       relay.I
	DbPath   string
	MaxLimit int
	Mux      *servemux.S
	*config.C
}

// NewServer initializes and returns a new Server instance based on the provided
// ServerParams and optional settings. It sets up storage, initializes the
// relay, and configures necessary components for server operation.
//
// # Parameters
//
// - sp (*ServerParams): The configuration parameters for initializing the
// server.
//
// - opts (...options.O): Optional settings that modify the server's behavior.
//
// # Return Values
//
// - s (*Server): The newly created Server instance.
//
// - err (error): An error if any step fails during initialization.
//
// # Expected Behaviour
//
// - Initializes storage with the provided database path.
//
// - Configures the server's options using the default settings and applies any
// optional settings provided.
//
// - Sets up a ServeMux for handling HTTP requests.
//
// - Initializes the relay, starting its operation in a separate goroutine.
func NewServer(
	sp *ServerParams, serveMux *servemux.S, opts ...options.O,
) (s *Server, err error) {
	op := options.Default()
	for _, opt := range opts {
		opt(op)
	}
	if storage := sp.Rl.Storage(); storage != nil {
		if err = storage.Init(sp.DbPath); chk.T(err) {
			return nil, fmt.Errorf("storage init: %w", err)
		}
	}
	s = &Server{
		Ctx:     sp.Ctx,
		Cancel:  sp.Cancel,
		relay:   sp.Rl,
		mux:     serveMux,
		options: op,
		C:       sp.C,
		Lists:   new(Lists),
		Peers:   new(Peers),
	}
	// Parse blacklist pubkeys
	for _, v := range s.C.Blacklist {
		var pk []byte
		if pk, err = keys.DecodeNpubOrHex(v); chk.E(err) {
			continue
		}
		s.blacklistPubkeys = append(s.blacklistPubkeys, pk)
	}
	chk.E(
		s.Peers.Init(sp.C.PeerRelays, sp.C.RelaySecret),
	)
	s.listeners = publish.New(socketapi.New(s), openapi.NewPublisher(s))
	go func() {
		if err := s.relay.Init(); chk.E(err) {
			s.Shutdown()
		}
	}()
	return s, nil
}

// ServeHTTP handles incoming HTTP requests according to the standard Nostr
// protocol. It specifically processes WebSocket upgrades and
// "application/nostr+json" Accept headers.
//
// # Parameters
//
// - w (http.ResponseWriter): The response writer for sending responses.
//
// - r (*http.Request): The request object containing client's details and data.
//
// # Expected Behaviour
//
// - Checks if the request URL path is "/".
//
// - For WebSocket upgrades, calls handleWebsocket method.
//
// - If "Accept" header is "application/nostr+json", calls HandleRelayInfo
// method.
//
// - Logs the HTTP request details for non-standard requests.
//
// - For all other paths, delegates to the internal mux's ServeHTTP method.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := s.Config()
	remote := helpers.GetRemoteFromReq(r)
	var whitelisted bool
	if len(c.Whitelist) > 0 {
		for _, addr := range c.Whitelist {
			if strings.HasPrefix(remote, addr) {
				whitelisted = true
			}
		}
	} else {
		whitelisted = true
	}
	if !whitelisted {
		return
	}
	// standard nostr protocol only governs the "root" path of the relay and
	// websockets
	if r.URL.Path == "/" {
		if r.Header.Get("Upgrade") == "websocket" {
			s.handleWebsocket(w, r)
			return
		}
		if r.Header.Get("Accept") == "application/nostr+json" {
			s.HandleRelayInfo(w, r)
			return
		}
	}
	log.I.F(
		"http request: %s from %s",
		r.URL.String(), helpers.GetRemoteFromReq(r),
	)
	s.mux.ServeHTTP(w, r)
}

// Start initializes the server by setting up a TCP listener and serving HTTP
// requests.
//
// # Parameters
//
// - host (string): The hostname or IP address to listen on.
//
// - port (int): The port number to bind to.
//
// - started (...chan bool): Optional channels that are closed after the server
// starts successfully.
//
// # Return Values
//
// - err (error): An error if any step fails during the server startup process.
//
// # Expected Behaviour
//
// - Joins the host and port into a full address string.
//
// - Logs the intention to start the relay listener at the specified address.
//
// - Listens for TCP connections on the specified address.
//
// - Configures an HTTP server with CORS middleware, sets timeouts, and binds it
// to the listener.
//
// - If any started channels are provided, closes them upon successful startup.
//
// - Starts serving requests using the configured HTTP server.
func (s *Server) Start(
	host string, port int, started ...chan bool,
) (err error) {
	log.I.F("running spider every %v", s.C.SpiderTime)
	if len(s.C.Owners) > 0 {
		// start up spider
		if err = s.Spider(s.C.Private); chk.E(err) {
			// there wasn't any owners, or they couldn't be found on the spider
			// seeds.
			err = nil
		}
	}
	// start up a spider run to trigger every 30 minutes
	ticker := time.NewTicker(s.C.SpiderTime)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err = s.Spider(s.C.Private); chk.E(err) {
					// there wasn't any owners, or they couldn't be found on the spider
					// seeds.
					err = nil
				}
			case <-s.Ctx.Done():
				log.I.F("stopping spider ticker")
				return
			}
		}
	}()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	log.I.F("starting relay listener at %s", addr)
	var ln net.Listener
	if ln, err = net.Listen("tcp", addr); err != nil {
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

// Shutdown gracefully shuts down the server and its components. It ensures that
// all resources are properly released.
//
// # Expected Behaviour
//
// - Logs shutting down message.
//
// - Cancels the context to stop ongoing operations.
//
// - Closes the event store, logging the action and checking for errors.
//
// - Shuts down the HTTP server, logging the action and checking for errors.
//
// - If the relay implements ShutdownAware, it calls OnShutdown with the
// context.
func (s *Server) Shutdown() {
	log.I.Ln("shutting down relay")
	s.Cancel()
	log.W.Ln("closing event store")
	chk.E(s.relay.Storage().Close())
	if s.httpServer != nil {
		log.W.Ln("shutting down relay listener")
		chk.E(s.httpServer.Shutdown(s.Ctx))
	}
	if f, ok := s.relay.(relay.ShutdownAware); ok {
		f.OnShutdown(s.Ctx)
	}
}

// Router retrieves and returns the HTTP ServeMux associated with the server.
//
// # Return Values
//
// - router (*http.ServeMux): The ServeMux instance used for routing HTTP
// requests.
//
// # Expected Behaviour
//
// - Returns the ServeMux that handles incoming HTTP requests to the server.
func (s *Server) Router() (router *http.ServeMux) {
	return s.mux.ServeMux
}
