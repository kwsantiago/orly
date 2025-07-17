package socketapi

import (
	"fmt"
	"orly.dev/pkg/encoders/envelopes"
	"orly.dev/pkg/encoders/envelopes/authenvelope"
	"orly.dev/pkg/encoders/envelopes/closeenvelope"
	"orly.dev/pkg/encoders/envelopes/eventenvelope"
	"orly.dev/pkg/encoders/envelopes/noticeenvelope"
	"orly.dev/pkg/encoders/envelopes/reqenvelope"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
)

// HandleMessage processes an incoming message, identifies its type, and
// delegates handling to the appropriate method based on the message's envelope
// type.
//
// Parameters:
//
//   - msg: A byte slice representing the raw message to be processed.
//
// Expected behavior:
//
// The method identifies the message type by examining its envelope label and
// passes the message payload to the corresponding handler function. If the type
// is unrecognized, it logs an error and generates an appropriate notice
// message. Handles errors in message identification or writing responses.
func (a *A) HandleMessage(msg []byte) {
	var notice []byte
	var err error
	var t string
	var rem []byte
	if t, rem, err = envelopes.Identify(msg); chk.E(err) {
		notice = []byte(err.Error())
	}
	switch t {
	case eventenvelope.L:
		notice = a.HandleEvent(a.Context(), rem, a.I)
	case reqenvelope.L:
		notice = a.HandleReq(
			a.Context(), rem,
			a.I,
		)
	case closeenvelope.L:
		notice = a.HandleClose(rem, a.I)
	case authenvelope.L:
		notice = a.HandleAuth(rem, a.I)
	default:
		notice = []byte(fmt.Sprintf("unknown envelope type %s\n%s", t, rem))
	}
	if len(notice) > 0 {
		log.D.F("notice->%s %s", a.RealRemote(), notice)
		if err = noticeenvelope.NewFrom(notice).Write(a.Listener); chk.E(err) {
			return
		}
	}

}
