package server

import (
	"errors"
	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/cors"
	"net"
	"net/http"
	"not.realy.lol/chk"
	"not.realy.lol/context"
	"not.realy.lol/log"
	"not.realy.lol/servemux"
	"not.realy.lol/store"
	"sync"
	"time"
)

type S struct {
	Ctx        context.T
	Cancel     context.F
	WG         sync.WaitGroup
	Addr       string
	mux        *servemux.S
	httpServer *http.Server
	Store      store.I
	huma.API
}

func (s *S) Init() {}

func (s *S) Start() (err error) {
	s.Init()
	var listener net.Listener
	if listener, err = net.Listen("tcp", s.Addr); chk.E(err) {
		return
	}
	s.httpServer = &http.Server{
		Handler:           cors.Default().Handler(s),
		Addr:              s.Addr,
		ReadHeaderTimeout: 7 * time.Second,
		IdleTimeout:       28 * time.Second,
	}
	log.I.F("listening on %s", s.Addr)
	if err = s.httpServer.Serve(listener); errors.Is(
		err, http.ErrServerClosed,
	) {
		return
	} else if chk.E(err) {
		return
	}
	return
}

// ServeHTTP is the server http.Handler.
func (s *S) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *S) Shutdown() {
	log.W.Ln("shutting down relay")
	s.Cancel()
	log.W.Ln("closing event store")
	chk.E(s.Store.Close())
	log.W.Ln("shutting down relay listener")
	chk.E(s.httpServer.Shutdown(s.Ctx))
}
