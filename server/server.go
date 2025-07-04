package server

import (
	"errors"
	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/cors"
	"net"
	"net/http"
	"not.realy.lol/chk"
	"not.realy.lol/config"
	"not.realy.lol/context"
	"not.realy.lol/helpers"
	"not.realy.lol/log"
	"not.realy.lol/servemux"
	"not.realy.lol/store"
	"sync"
	"time"
)

type S struct {
	Ctx        context.T
	Cancel     context.F
	WG         *sync.WaitGroup
	Addr       string
	Cfg        *config.C
	Mux        *servemux.S
	HTTPServer *http.Server
	Store      store.I
	huma.API
}

func (s *S) Init() {}

func (s *S) Start() (err error) {
	s.WG.Add(1)
	s.Init()
	var listener net.Listener
	if listener, err = net.Listen("tcp", s.Addr); chk.E(err) {
		return
	}
	s.HTTPServer = &http.Server{
		Handler:           cors.Default().Handler(s),
		Addr:              s.Addr,
		ReadHeaderTimeout: 7 * time.Second,
		IdleTimeout:       28 * time.Second,
	}
	if s.Cfg.DNS != "" {
		log.I.F("listening on %s http://%s", s.Cfg.DNS, s.Addr)
	} else {
		log.I.F("listening on http://%s\n", s.Addr)
	}
	if err = s.HTTPServer.Serve(listener); errors.Is(
		err, http.ErrServerClosed,
	) {
		err = nil
		return
	} else if chk.E(err) {
		return
	}
	return
}

// ServeHTTP is the server http.Handler.
func (s *S) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote := helpers.GetRemoteFromReq(r)
	log.T.F("server.S.ServeHTTP to %s", remote)
	s.Mux.ServeHTTP(w, r)
}

func (s *S) Shutdown() {
	log.W.Ln("shutting down relay")
	s.Cancel()
	// log.W.Ln("closing event store")
	// chk.E(s.Store.Close())
	log.W.Ln("shutting down relay listener")
	chk.E(s.HTTPServer.Shutdown(s.Ctx))
	s.WG.Done()
}
