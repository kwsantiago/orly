package socketapi

import (
	"fmt"
	"orly.dev/chk"
	"orly.dev/log"

	"orly.dev/envelopes"
	"orly.dev/envelopes/authenvelope"
	"orly.dev/envelopes/closeenvelope"
	"orly.dev/envelopes/eventenvelope"
	"orly.dev/envelopes/noticeenvelope"
	"orly.dev/envelopes/reqenvelope"
)

func (a *A) HandleMessage(msg []byte) {
	var notice []byte
	var err error
	var t string
	var rem []byte
	if t, rem, err = envelopes.Identify(msg); chk.E(err) {
		notice = []byte(err.Error())
	}
	// rl := a.Relay()
	switch t {
	case eventenvelope.L:
		notice = a.HandleEvent(a.Context(), rem, a.Server)
	case reqenvelope.L:
		notice = a.HandleReq(
			a.Context(), rem,
			// a.Options().SkipEventFunc,
			a.Server,
		)
	case closeenvelope.L:
		notice = a.HandleClose(rem, a.Server)
	case authenvelope.L:
		notice = a.HandleAuth(rem, a.Server)
	default:
		// if wsh, ok := rl.(relay.WebSocketHandler); ok {
		//	wsh.HandleUnknownType(a.Listener, t, rem)
		// } else {
		notice = []byte(fmt.Sprintf("unknown envelope type %s\n%s", t, rem))
		// }
	}
	if len(notice) > 0 {
		log.D.F("notice->%s %s", a.RealRemote(), notice)
		if err = noticeenvelope.NewFrom(notice).Write(a.Listener); err != nil {
			return
		}
	}

}
