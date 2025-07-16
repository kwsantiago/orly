package realy

import (
	"net/http"

	"orly.dev/protocol/socketapi"
)

func (s *Server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	a := &socketapi.A{S: s}
	a.Serve(w, r, s)
}
