package socketapi

import (
	"net/http"
	"not.realy.lol/context"
	"not.realy.lol/helpers"
	"not.realy.lol/interfaces/server"
	"not.realy.lol/log"
	"not.realy.lol/servemux"
)

type A struct {
	Ctx context.T
	server.I
	// Web is an optional web server that appears on `/` with no Upgrade for
	// websockets or Accept for application/nostr+json present.
	Web http.Handler
}

func New(s server.I, path string, sm *servemux.S) {
	a := &A{I: s}
	sm.Handle(path, a)
	return
}

// ServeHTTP handles incoming HTTP requests and processes them accordingly. It
// serves the relayinfo for specific headers or delegates to a web handler. It
// processes WebSocket upgrade requests when applicable.
func (a *A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote := helpers.GetRemoteFromReq(r)
	log.T.F("socketAPI handling %s", remote)
	if r.Header.Get("Upgrade") != "websocket" &&
		r.Header.Get("Accept") == "application/nostr+json" {
		log.T.F("serving realy info %s", remote)
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
}
