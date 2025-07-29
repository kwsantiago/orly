package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"os"
)

type NostrJSON struct {
	Names  map[string]string   `json:"names"`
	Relays map[string][]string `json:"relays"`
}

// NostrDNS handles the configuration and registration of a Nostr DNS endpoint
// for a given hostname and backend address.
//
// # Parameters
//
// - hn (string): The hostname for which the Nostr DNS entry is being configured.
//
// - ba (string): The path to the JSON file containing the Nostr DNS data.
//
// - mux (*http.ServeMux): The HTTP serve multiplexer to which the Nostr DNS
// handler will be registered.
//
// # Return Values
//
// - err (error): An error if any step fails during the configuration or
// registration process.
//
// # Expected behaviour
//
// - Reads the JSON file specified by `ba` and parses its contents into a
// NostrJSON struct.
//
// - Registers a new HTTP handler on the provided `mux` for the
// `.well-known/nostr.json` endpoint under the specified hostname.
//
// - The handler serves the parsed Nostr DNS data with appropriate HTTP headers
// set for CORS and content type.
func NostrDNS(hn, ba string, mux *http.ServeMux) (err error) {
	log.T.Ln(hn, ba)
	var fb []byte
	if fb, err = os.ReadFile(ba); chk.E(err) {
		return
	}
	var v NostrJSON
	if err = json.Unmarshal(fb, &v); chk.E(err) {
		return
	}
	var jb []byte
	if jb, err = json.Marshal(v); chk.E(err) {
		return
	}
	nostrJSON := string(jb)
	mux.HandleFunc(
		hn+"/.well-known/nostr.json",
		func(writer http.ResponseWriter, request *http.Request) {
			log.T.Ln("serving nostr json to", hn)
			writer.Header().Set(
				"Access-Control-Allow-Methods",
				"GET,HEAD,PUT,PATCH,POST,DELETE",
			)
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			writer.Header().Set("Content-Type", "application/json")
			writer.Header().Set(
				"Content-Length", fmt.Sprint(len(nostrJSON)),
			)
			writer.Header().Set(
				"strict-transport-security",
				"max-age=0; includeSubDomains",
			)
			fmt.Fprint(writer, nostrJSON)
		},
	)
	return
}
