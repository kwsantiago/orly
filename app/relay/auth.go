package relay

import (
	"net/http"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
	"orly.dev/utils/lol"
	"strconv"
	"strings"
)

// ServiceURL returns the address of the relay to send back in auth responses.
// If auth is disabled, this returns an empty string.
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
