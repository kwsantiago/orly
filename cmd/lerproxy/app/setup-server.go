package app

import (
	"fmt"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"orly.dev/cmd/lerproxy/utils"
	"orly.dev/pkg/utils/chk"
	"os"
)

// SetupServer configures and returns an HTTP server instance with proxy
// handling and automatic certificate management based on the provided RunArgs
// configuration.
//
// # Parameters
//
// - a (RunArgs): The configuration arguments containing settings for the server
// address, cache directory, mapping file, HSTS header, email, and certificates.
//
// # Return Values
//
// - s (*http.Server): The configured HTTP server instance.
//
// - h (http.Handler): The HTTP handler used for proxying requests and managing
// automatic certificate challenges.
//
// - err (error): An error if any step during setup fails.
//
// # Expected behaviour
//
// - Reads the hostname to backend address mapping from the specified
// configuration file.
//
// - Sets up a proxy handler that routes incoming requests based on the defined
// mappings.
//
// - Enables HSTS header support if enabled in the RunArgs.
//
// - Creates the cache directory for storing certificates and keys if it does not
// already exist.
//
// - Configures an autocert.Manager to handle automatic certificate management,
// including hostname whitelisting, email contact, and cache storage.
//
// - Initializes the HTTP server with proxy handler, address, and TLS
// configuration.
func SetupServer(a RunArgs) (s *http.Server, h http.Handler, err error) {
	var mapping map[string]string
	if mapping, err = ReadMapping(a.Conf); chk.E(err) {
		return
	}
	var proxy http.Handler
	if proxy, err = SetProxy(mapping); chk.E(err) {
		return
	}
	if a.HSTS {
		proxy = &Proxy{Handler: proxy}
	}
	if err = os.MkdirAll(a.Cache, 0700); chk.E(err) {
		err = fmt.Errorf(
			"cannot create cache directory %q: %v",
			a.Cache, err,
		)
		chk.E(err)
		return
	}
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(a.Cache),
		HostPolicy: autocert.HostWhitelist(utils.GetKeys(mapping)...),
		Email:      a.Email,
	}
	s = &http.Server{
		Handler:   proxy,
		Addr:      a.Addr,
		TLSConfig: TLSConfig(&m, a.Certs...),
	}
	h = m.HTTPHandler(nil)
	return
}
