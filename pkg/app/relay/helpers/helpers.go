package helpers

import (
	"net/http"
	"strings"
)

// GenerateDescription generates a detailed description containing the provided
// text and an optional list of scopes.
//
// # Parameters
//
//   - text: A string representing the base description.
//
//   - scopes: A slice of strings indicating scopes to be included in the
//     description.
//
// # Return Values
//
//   - A string combining the base description and a formatted list of
//     scopes, if provided.
//
// # Expected behaviour
//
// The function appends a formatted list of scopes to the base description if
// any scopes are provided. If no scopes are provided, it returns the base
// description unchanged. The formatted list of scopes includes each scope
// surrounded by backticks and separated by commas.
func GenerateDescription(text string, scopes []string) string {
	if len(scopes) == 0 {
		return text
	}
	result := make([]string, 0)
	for _, value := range scopes {
		result = append(result, "`"+value+"`")
	}
	return text + "<br/><br/>**Scopes**<br/>" + strings.Join(result, ", ")
}

// GetRemoteFromReq retrieves the originating IP address of the client from
// an HTTP request, considering standard and non-standard proxy headers.
//
// # Parameters
//
//   - r: The HTTP request object containing details of the client and
//     routing information.
//
// # Return Values
//
//   - rr: A string value representing the IP address of the originating
//     remote client.
//
// # Expected behaviour
//
// The function first checks for the standardized "Forwarded" header (RFC 7239)
// to identify the original client IP. If that isn't available, it falls back to
// the "X-Forwarded-For" header. If both headers are absent, it defaults to
// using the request's RemoteAddr.
//
// For the "Forwarded" header, it extracts the client IP from the "for"
// parameter. For the "X-Forwarded-For" header, if it contains one IP, it
// returns that. If it contains two IPs, it returns the second.
func GetRemoteFromReq(r *http.Request) (rr string) {
	// First check for the standardized Forwarded header (RFC 7239)
	forwarded := r.Header.Get("Forwarded")
	if forwarded != "" {
		// Parse the Forwarded header which can contain multiple parameters
		//
		// Format:
		//
		// 	Forwarded: by=<identifier>;for=<identifier>;host=<host>;proto=<http|https>
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "for=") {
				// Extract the client IP from the "for" parameter
				forValue := strings.TrimPrefix(part, "for=")
				// Remove quotes if present
				forValue = strings.Trim(forValue, "\"")
				// Handle IPv6 addresses which are enclosed in square brackets
				forValue = strings.Trim(forValue, "[]")
				return forValue
			}
		}
	}
	// If the Forwarded header is not available or doesn't contain "for"
	// parameter, fall back to X-Forwarded-For
	rem := r.Header.Get("X-Forwarded-For")
	if rem == "" {
		rr = r.RemoteAddr
	} else {
		splitted := strings.Split(rem, " ")
		if len(splitted) == 1 {
			rr = splitted[0]
		}
		if len(splitted) == 2 {
			rr = splitted[1]
		}
	}
	return
}
