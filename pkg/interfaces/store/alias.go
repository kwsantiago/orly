package store

import (
	"net/http"
	"orly.dev/pkg/encoders/envelopes/okenvelope"
	"orly.dev/pkg/encoders/subscription"
)

type SubID = subscription.Id
type Responder = http.ResponseWriter
type Req = *http.Request
type OK = okenvelope.T
