package store

import (
	"net/http"

	"orly.dev/encoders/envelopes/okenvelope"
	"orly.dev/encoders/subscription"
)

type SubID = subscription.Id
type Responder = http.ResponseWriter
type Req = *http.Request
type OK = okenvelope.T
