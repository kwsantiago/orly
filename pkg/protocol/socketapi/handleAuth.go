package socketapi

import (
	"orly.dev/pkg/encoders/envelopes/authenvelope"
	"orly.dev/pkg/encoders/envelopes/okenvelope"
	"orly.dev/pkg/encoders/reason"
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/protocol/auth"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/iptracker"
	"orly.dev/pkg/utils/log"
)

// HandleAuth processes authentication data received from a remote client,
// validates it against the server's challenge, and sets up authentication if
// successful.
//
// # Parameters
//
// - b ([]byte): The raw byte slice containing the authentication response to be
// processed.
//
// - srv (server.I): A reference to the server interface that provides context
// for the authentication process.
//
// # Return Values
//
// - msg ([]byte): An optional message returned if the authentication fails or
// requires further action.
//
// # Expected behaviour
//
// Handles the authentication process by checking if authentication is required,
// unmarshalling and validating the response against a challenge, logging
// relevant information, and setting up the authenticated state on successful
// validation.
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
				env.Event.ID, true,
			).Write(a.Listener); chk.E(err) {
				return
			}
			log.D.F(
				"%s authed to pubkey,%0x", a.Listener.RealRemote(),
				env.Event.Pubkey,
			)
			a.Listener.SetAuthedPubkey(env.Event.Pubkey)
			
			// If authentication is successful, remove any blocks for this IP
			iptracker.Global.Authenticate(a.Listener.RealRemote())
		}
	}
	return
}
