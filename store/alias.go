package store

import (
	"net/http"

	"not.realy.lol/envelopes/okenvelope"
	"not.realy.lol/subscription"
)

type SubID = subscription.Id
type Responder = http.ResponseWriter
type Req = *http.Request
type OK = okenvelope.T
