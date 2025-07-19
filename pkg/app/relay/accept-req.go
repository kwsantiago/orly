package relay

import (
	"net/http"

	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/utils/context"
)

// AcceptReq determines whether a request should be accepted based on
// authentication and public readability settings.
//
// Parameters:
//   - c: context for the request handling
//   - hr: HTTP request received
//   - f: filters to apply
//   - authedPubkey: authenticated public key (if any)
//   - remote: remote address of the request
//
// Return Values:
//   - allowed: filters that are allowed after processing
//   - accept: boolean indicating whether the request should be accepted
//   - modified: boolean indicating if the request has been modified during
//     processing
//
// Expected Behaviour:
//
// - If authentication is required and there's no authenticated public key,
// reject the request.
//
// - Otherwise, accept the request.
func (s *Server) AcceptReq(
	c context.T, hr *http.Request, f *filters.T,
	authedPubkey []byte, remote string,
) (allowed *filters.T, accept bool, modified bool) {
	// if auth is required, and not public readable, reject
	if s.AuthRequired() && len(authedPubkey) == 0 && !s.PublicReadable() {
		return
	}
	accept = true
	return
}
