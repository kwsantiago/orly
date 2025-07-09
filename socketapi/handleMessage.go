package socketapi

import (
	"fmt"

	"orly.dev/chk"
	"orly.dev/envelopes"
	"orly.dev/envelopes/closeenvelope"
	"orly.dev/envelopes/eventenvelope"
	"orly.dev/envelopes/noticeenvelope"
	"orly.dev/envelopes/reqenvelope"
	"orly.dev/log"
)

func (a *A) HandleMessage(msg []byte, remote string) {
	log.T.F("received message from %s\n%s", remote, msg)
	var notice []byte
	var err error
	var t string
	var rem []byte
	if t, rem = envelopes.Identify(msg); chk.E(err) {
		notice = []byte(err.Error())
	}
	switch t {
	case eventenvelope.L:
		notice = a.HandleEvent(rem, a.I, remote)
	case reqenvelope.L:
		notice = a.HandleReq(
			rem, a.I, remote,
		)
	case closeenvelope.L:
		notice = a.HandleClose(rem, a.I)
	// case authenvelope.L:
	// notice = a.HandleAuth(rem, a.Server)
	default:
		notice = []byte(fmt.Sprintf("unknown envelope type %s\n%s", t, rem))
	}
	if len(notice) > 0 {
		log.D.F("notice->%s %s", remote, notice)
		if err = noticeenvelope.NewFrom(notice).Write(a.Listener); err != nil {
			return
		}
	}
}
