package store

import (
	"net/http"

	"orly.dev/envelopes/okenvelope"
	"orly.dev/subscription"
)

type SubID = subscription.Id
type Responder = http.ResponseWriter
type Req = *http.Request
type OK = okenvelope.T
