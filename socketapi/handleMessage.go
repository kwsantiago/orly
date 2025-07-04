package socketapi

import (
	"fmt"

	"not.realy.lol/chk"
	"not.realy.lol/envelopes"
	"not.realy.lol/envelopes/authenvelope"
	"not.realy.lol/envelopes/closeenvelope"
	"not.realy.lol/envelopes/eventenvelope"
	"not.realy.lol/envelopes/noticeenvelope"
	"not.realy.lol/envelopes/reqenvelope"
	"not.realy.lol/log"
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
		// notice = a.HandleReq(
		// 	a.Context(), rem, a.Server, a.Listener.AuthedBytes(), remote,
		// )
	case closeenvelope.L:
		// notice = a.HandleClose(rem, a.Server)
	case authenvelope.L:
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
