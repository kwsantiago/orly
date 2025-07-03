package server

import (
	"net/http"
	"not.realy.lol/context"
	"not.realy.lol/servemux"
	"sync"
)

type S struct {
	Ctx        context.T
	Cancel     context.F
	WG         sync.WaitGroup
	Addr       string
	mux        *servemux.S
	httpServer *http.Server
}
