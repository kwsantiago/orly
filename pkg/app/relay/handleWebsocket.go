package relay

import (
	"net/http"
	"orly.dev/pkg/protocol/socketapi"
)

func (s *Server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	a := &socketapi.A{I: s}
	a.Serve(w, r, s)
}
