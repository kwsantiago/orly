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
}

func New(s server.I, path string, sm *servemux.S) {
	a := &A{I: s}
	sm.Handle(path, a)
	return
}

func (a *A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote := helpers.GetRemoteFromReq(r)
	log.T.F("socketAPI handling %s", remote)
	if r.Header.Get("Upgrade") != "websocket" && r.Header.Get("Accept") == "application/nostr+json" {
		log.T.F("serving realy info %s", remote)
		a.I.HandleRelayInfo(w, r)
		return
	}
	if r.Header.Get("Upgrade") != "websocket" {
		// for now just serve relay info on the /
		a.I.HandleRelayInfo(w, r)
		// // todo: we can put a website here
		// 	http.Error(
		// 		w, http.StatusText(http.StatusUpgradeRequired),
		// 		http.StatusUpgradeRequired,
		// 	)
		return
	}
}
