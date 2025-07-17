package relay

import (
	"net/http"
	"strconv"
	"strings"

	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
)

// ServiceURL determines the service URL based on the request headers.
//
// Parameters:
//
// - req: The HTTP request object containing header information.
//
// Return Values:
//
// - st: A string representing the constructed service URL.
//
// Expected Behaviour:
//
// - Checks if authentication is required. If not, returns an empty string.
//
// - Retrieves the host from the "X-Forwarded-Host" header or uses the request's
// Host if not present.
//
// - Determines the protocol (ws or wss) based on the "X-Forwarded-Proto" header
// or other conditions like the presence of a port or IP address.
//
// - Constructs and returns the service URL in the format "protocol://host".
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
