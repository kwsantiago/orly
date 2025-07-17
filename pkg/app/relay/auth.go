package relay

import (
	"net/http"
	"strconv"
	"strings"

	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
)

// ServiceURL
//
// Parameters
//
// - req: HTTP request containing headers for determining service URL
//
// Return Values
//
// - st: Constructed service URL string
//
// Expected Behaviour
//
// Builds a WebSocket URL based on request headers and host information. If
// authentication is not required, logs and returns immediately. Determines
// protocol (ws/wss) using X-Forwarded-Proto header or host characteristics.
// Constructs URL with determined protocol and host.
func (s *Server) ServiceURL(req *http.Request) (st string) {
	lol.Tracer("ServiceURL")
	defer func() { lol.Tracer("end ServiceURL", st) }()
	if !s.AuthRequired() {
		log.T.F("auth not required")
		return
	}
	host := req.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = req.Host
	}
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if host == "localhost" {
			proto = "ws"
		} else if strings.Contains(host, ":") {
			// has a port number
			proto = "ws"
		} else if _, err := strconv.Atoi(
			strings.ReplaceAll(
				host, ".",
				"",
			),
		); chk.E(err) {
			// it's a naked IP
			proto = "ws"
		} else {
			proto = "wss"
		}
	} else if proto == "https" {
		proto = "wss"
	} else if proto == "http" {
		proto = "ws"
	}
	return proto + "://" + host
}
