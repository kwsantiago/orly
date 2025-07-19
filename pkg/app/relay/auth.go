package relay

import (
	"net/http"
	"strconv"
	"strings"

	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
)

// ServiceURL constructs the service URL based on the incoming HTTP request. It
// checks for authentication requirements and determines the protocol (ws or
// wss) based on headers like X-Forwarded-Host, X-Forwarded-Proto, and the host
// itself.
//
// Parameters:
//
// - req: A pointer to an http.Request object representing the incoming request.
//
// Return Values:
//
// - st: A string representing the constructed service URL.
//
// Expected Behaviour:
//
// - Checks if authentication is required.
//
// - Retrieves the host from X-Forwarded-Host or falls back to req.Host.
//
// - Determines the protocol (ws or wss) based on various conditions including
// headers and host details.
//
// - Returns the constructed URL string.
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
