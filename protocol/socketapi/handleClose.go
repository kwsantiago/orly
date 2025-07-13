package socketapi

import (
	"orly.dev/app/realy/interfaces"
	"orly.dev/encoders/envelopes/closeenvelope"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
)

func (a *A) HandleClose(
	req []byte,
	srv interfaces.Server,
) (note []byte) {
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
	srv.Publisher().Receive(
		&W{
			Cancel:   true,
			Listener: a.Listener,
			Id:       env.ID.String(),
		},
	)
	return
}
