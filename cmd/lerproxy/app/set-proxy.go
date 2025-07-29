package app

import (
	"fmt"
	"io"
	log2 "log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SetProxy creates an HTTP handler that routes incoming requests to specified
// backend addresses based on hostname mappings.
//
// # Parameters
//
// - mapping (map[string]string): A map where keys are hostnames and values are
// the corresponding backend addresses.
//
// # Return Values
//
// - h (http.Handler): The HTTP handler configured with the proxy settings.
// - err (error): An error if the mapping is empty or invalid.
//
// # Expected behaviour
//
// - Validates that the provided hostname to backend address mapping is not empty.
//
// - Creates a new ServeMux and configures it to route requests based on the
// specified hostnames and backend addresses.
//
// - Handles special cases such as vanity URLs, Nostr DNS entries, and Unix
// socket connections.
func SetProxy(mapping map[string]string) (h http.Handler, err error) {
	if len(mapping) == 0 {
		return nil, fmt.Errorf("empty mapping")
	}
	mux := http.NewServeMux()
	for hostname, backendAddr := range mapping {
		hn, ba := hostname, backendAddr
		if strings.ContainsRune(hn, os.PathSeparator) {
			err = log.E.Err("invalid hostname: %q", hn)
			return
		}
		network := "tcp"
		if ba != "" && ba[0] == '@' && runtime.GOOS == "linux" {
			// append \0 to address so addrlen for connect(2) is calculated in a
			// way compatible with some other implementations (i.e. uwsgi)
			network, ba = "unix", ba+string(byte(0))
		} else if strings.HasPrefix(ba, "git+") {
			GoVanity(hn, ba, mux)
			continue
		} else if filepath.IsAbs(ba) {
			network = "unix"
			switch {
			case strings.HasSuffix(ba, string(os.PathSeparator)):
				// path specified as directory with explicit trailing slash; add
				// this path as static site
				fs := http.FileServer(http.Dir(ba))
				mux.Handle(hn+"/", fs)
				continue
			case strings.HasSuffix(ba, "nostr.json"):
				if err = NostrDNS(hn, ba, mux); err != nil {
					continue
				}
				continue
			}
		} else if u, err := url.Parse(ba); err == nil {
			switch u.Scheme {
			case "http", "https":
				rp := NewSingleHostReverseProxy(u)
				modifyCORSResponse := func(res *http.Response) error {
					res.Header.Set(
						"Access-Control-Allow-Methods",
						"GET,HEAD,PUT,PATCH,POST,DELETE",
					)
					// res.Header.Set("Access-Control-Allow-Credentials", "true")
					res.Header.Set("Access-Control-Allow-Origin", "*")
					return nil
				}
				rp.ModifyResponse = modifyCORSResponse
				rp.ErrorLog = log2.New(
					os.Stderr, "lerproxy", log2.Llongfile,
				)
				rp.BufferPool = Pool{}
				mux.Handle(hn+"/", rp)
				continue
			}
		}
		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = req.Host
				req.Header.Set("X-Forwarded-Proto", "https")
				req.Header.Set("X-Forwarded-For", req.RemoteAddr)
				req.Header.Set(
					"Access-Control-Allow-Methods",
					"GET,HEAD,PUT,PATCH,POST,DELETE",
				)
				req.Header.Set("Access-Control-Allow-Origin", "*")
				log.D.Ln(req.URL, req.RemoteAddr)
			},
			Transport: &http.Transport{
				DialContext: func(c context.T, n, addr string) (
					net.Conn, error,
				) {
					return net.DialTimeout(network, ba, 5*time.Second)
				},
			},
			ErrorLog:   log2.New(io.Discard, "", 0),
			BufferPool: Pool{},
		}
		mux.Handle(hn+"/", rp)
	}
	return mux, nil
}
