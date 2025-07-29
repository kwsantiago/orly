package app

import (
	"fmt"
	"net/http"
	"orly.dev/pkg/utils/log"
	"strings"
)

// GoVanity configures an HTTP handler for redirecting requests to vanity URLs
// based on the provided hostname and backend address.
//
// # Parameters
//
// - hn (string): The hostname associated with the vanity URL.
//
// - ba (string): The backend address, expected to be in the format
// "git+<repository-path>".
//
// - mux (*http.ServeMux): The HTTP serve multiplexer where the handler will be
// registered.
//
// # Expected behaviour
//
// - Splits the backend address to extract the repository path from the "git+" prefix.
//
// - If the split fails, logs an error and returns without registering a handler.
//
// - Generates an HTML redirect page containing metadata for Go import and
// redirects to the extracted repository path.
//
// - Registers a handler on the provided ServeMux that serves this redirect page
// when requests are made to the specified hostname.
func GoVanity(hn, ba string, mux *http.ServeMux) {
	split := strings.Split(ba, "git+")
	if len(split) != 2 {
		log.E.Ln("invalid go vanity redirect: %s: %s", hn, ba)
		return
	}
	redirector := fmt.Sprintf(
		`<html><head><meta name="go-import" content="%s git %s"/><meta http-equiv = "refresh" content = " 3 ; url = %s"/></head><body>redirecting to <a href="%s">%s</a></body></html>`,
		hn, split[1], split[1], split[1], split[1],
	)
	mux.HandleFunc(
		hn+"/",
		func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set(
				"Access-Control-Allow-Methods",
				"GET,HEAD,PUT,PATCH,POST,DELETE",
			)
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			writer.Header().Set("Content-Type", "text/html")
			writer.Header().Set(
				"Content-Length", fmt.Sprint(len(redirector)),
			)
			writer.Header().Set(
				"strict-transport-security",
				"max-age=0; includeSubDomains",
			)
			fmt.Fprint(writer, redirector)
		},
	)
}
