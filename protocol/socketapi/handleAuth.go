package socketapi

import (
	"orly.dev/encoders/envelopes/authenvelope"
	"orly.dev/encoders/envelopes/okenvelope"
	"orly.dev/encoders/reason"
	"orly.dev/interfaces/server"
	"orly.dev/protocol/auth"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
)

func (a *A) HandleAuth(b []byte, srv server.I) (msg []byte) {
	if a.I.AuthRequired() {
		log.I.F("AUTH:\n%s", b)
		var err error
		var rem []byte
		env := authenvelope.NewResponse()
		if rem, err = env.Unmarshal(b); chk.E(err) {
			return
		}
		if len(rem) > 0 {
			log.I.F("extra '%s'", rem)
		}
		var valid bool
		if valid, err = auth.Validate(
			env.Event, a.Listener.Challenge(),
			srv.ServiceURL(a.Listener.Request),
		); err != nil {
			e := err.Error()
			if err = Ok.Error(a, env, e); chk.E(err) {
				return []byte(e)
			}
			return reason.Error.F(e)
		} else if !valid {
			if err = Ok.Error(a, env, "failed to authenticate"); chk.E(err) {
				return
			}
			return reason.Error.F("auth response does not validate")
		} else {
			if err = okenvelope.NewFrom(
				env.Event.Id, true,
			).Write(a.Listener); chk.E(err) {
				return
			}
			log.D.F(
				"%s authed to pubkey,%0x", a.Listener.RealRemote(),
				env.Event.Pubkey,
			)
			a.Listener.SetAuthedPubkey(env.Event.Pubkey)
		}
	}
	return
}
