package socketapi

import (
	"orly.dev/chk"
	"orly.dev/envelopes/closeenvelope"
	"orly.dev/interfaces/server"
	"orly.dev/log"
	"orly.dev/publish"
)

func (a *A) HandleClose(req []byte, srv server.I) (note []byte) {
	var err error
	var rem []byte
	env := closeenvelope.New()
	if rem, err = env.Unmarshal(req); chk.E(err) {
		return []byte(err.Error())
	}
	if len(rem) > 0 {
		log.I.F("extra '%s'", rem)
	}
	if env.ID.String() == "" {
		return []byte("CLOSE has no <id>")
	}
	publish.P.Receive(
		&W{
			Cancel: true,
			I:      a.Listener,
			Id:     env.ID.String(),
		},
	)
	return
}
