package app

import (
	"crypto/tls"
	"golang.org/x/crypto/acme/autocert"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"strings"
	"sync"
)

// TLSConfig creates a custom TLS configuration that combines automatic
// certificate management with explicitly provided certificates.
//
// # Parameters
//
// - m (*autocert.Manager): The autocert manager used for managing automatic
// certificate generation and retrieval.
//
// - certs (...string): A variadic list of certificate definitions in the format
// "domain:/path/to/cert", where each domain maps to a certificate file. The
// corresponding key file is expected to be at "/path/to/cert.key".
//
// # Return Values
//
// - tc (*tls.Config): A new TLS configuration that prioritises explicitly
// provided certificates over automatically generated ones.
//
// # Expected behaviour
//
// - Loads all explicitly provided certificates and maps them to their
// respective domains.
//
// - Creates a custom GetCertificate function that checks if the requested
// domain matches any of the explicitly provided certificates, returning those
// first.
//
// - Falls back to the autocert manager's GetCertificate method if no explicit
// certificate is found for the requested domain.
func TLSConfig(m *autocert.Manager, certs ...string) (tc *tls.Config) {
	certMap := make(map[string]*tls.Certificate)
	var mx sync.Mutex
	for _, cert := range certs {
		split := strings.Split(cert, ":")
		if len(split) != 2 {
			log.E.F("invalid certificate parameter format: `%s`", cert)
			continue
		}
		var err error
		var c tls.Certificate
		if c, err = tls.LoadX509KeyPair(
			split[1]+".crt", split[1]+".key",
		); chk.E(err) {
			continue
		}
		certMap[split[0]] = &c
	}
	tc = m.TLSConfig()
	tc.GetCertificate = func(helo *tls.ClientHelloInfo) (
		cert *tls.Certificate, err error,
	) {
		mx.Lock()
		var own string
		for i := range certMap {
			// to also handle explicit subdomain certs, prioritize over a root
			// wildcard.
			if helo.ServerName == i {
				own = i
				break
			}
			// if it got to us and ends in the same-name dot tld assume the
			// subdomain was redirected, or it is a wildcard certificate; thus
			// only the ending needs to match.
			if strings.HasSuffix(helo.ServerName, i) {
				own = i
				break
			}
		}
		if own != "" {
			defer mx.Unlock()
			return certMap[own], nil
		}
		mx.Unlock()
		return m.GetCertificate(helo)
	}
	return
}
