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

// HandleMessage processes an incoming byte slice message by identifying its type
// and routing it to the appropriate handler method, generating and sending a
// notice response if necessary.
//
// # Parameters
//
// - msg ([]byte): The incoming message data to be processed.
//
// # Expected behaviour
//
// Processes the message by identifying its envelope type, routes it to the
// corresponding handler method, generates a notice for errors or unknown types,
// logs the notice, and writes it back to the listener if required.
func (a *A) HandleMessage(msg, authedPubkey []byte) {
	remote := a.Listener.RealRemote()
	log.T.C(
		func() string {
			return fmt.Sprintf(
				"%s received message:\n%s", remote, string(msg),
			)
		},
	)
	var notice []byte
	var err error
	var t string
	var rem []byte
	if t, rem, err = envelopes.Identify(msg); chk.E(err) {
		notice = []byte(err.Error())
	}
	switch t {
	case eventenvelope.L:
		notice = a.HandleEvent(a.Ctx, rem, a.I)
	case reqenvelope.L:
		notice = a.HandleReq(a.Ctx, rem, a.I)
	case closeenvelope.L:
		notice = a.HandleClose(rem, a.I)
	case authenvelope.L:
		notice = a.HandleAuth(rem, a.I)
	default:
		notice = []byte(fmt.Sprintf("unknown envelope type %s\n%s", t, rem))
	}
	if len(notice) > 0 {
		log.D.C(
			func() string {
				return fmt.Sprintf(
					"notice->%s %s", a.RealRemote(), notice,
				)
			},
		)
		if err = noticeenvelope.NewFrom(notice).Write(a.Listener); chk.E(err) {
			return
		}
	}

}
