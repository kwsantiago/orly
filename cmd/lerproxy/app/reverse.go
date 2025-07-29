package app

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"orly.dev/cmd/lerproxy/utils"
	"orly.dev/pkg/utils/log"
)

// NewSingleHostReverseProxy is a copy of httputil.NewSingleHostReverseProxy
// with the addition of forwarding headers:
//
// - Legacy X-Forwarded-* headers (X-Forwarded-Proto, X-Forwarded-For, X-Forwarded-Host)
//
// - Standardized Forwarded header according to RFC 7239
// (https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Forwarded)
func NewSingleHostReverseProxy(target *url.URL) (rp *httputil.ReverseProxy) {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		log.D.S(req)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = utils.SingleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
		// Set X-Forwarded-* headers for backward compatibility
		req.Header.Set("X-Forwarded-Proto", "https")
		// Get client IP address
		clientIP := req.RemoteAddr
		if fwdFor := req.Header.Get("X-Forwarded-For"); fwdFor != "" {
			clientIP = fwdFor + ", " + clientIP
		}
		req.Header.Set("X-Forwarded-For", clientIP)
		// Set X-Forwarded-Host if not already set
		if _, exists := req.Header["X-Forwarded-Host"]; !exists {
			req.Header.Set("X-Forwarded-Host", req.Host)
		}
		// Set standardized Forwarded header according to RFC 7239
		// Format: Forwarded: by=<identifier>;for=<identifier>;host=<host>;proto=<http|https>
		forwardedProto := "https"
		forwardedHost := req.Host
		forwardedFor := clientIP
		// Build the Forwarded header value
		forwardedHeader := "proto=" + forwardedProto
		if forwardedFor != "" {
			forwardedHeader += ";for=" + forwardedFor
		}
		if forwardedHost != "" {
			forwardedHeader += ";host=" + forwardedHost
		}
		req.Header.Set("Forwarded", forwardedHeader)
	}
	rp = &httputil.ReverseProxy{Director: director}
	return
}
