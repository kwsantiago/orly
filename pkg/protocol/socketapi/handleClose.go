package socketapi

import (
	"orly.dev/pkg/encoders/envelopes/closeenvelope"
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
)

// HandleClose processes a CLOSE envelope by unmarshalling the request,
// validates the presence of an <id> field, and signals cancellation for
// the associated listener through the server's publisher mechanism.
//
// # Parameters
//
//   - req ([]byte): The raw byte slice containing the CLOSE envelope data.
//
//   - srv (server.I): A reference to the server interface used to access
//     publishing capabilities.
//
// # Return Values
//
//   - note ([]byte): An empty byte slice if successful, or an error message
//     if the envelope is invalid or missing required fields.
//
// # Expected behaviour
//
// Processes the CLOSE envelope by unmarshalling it into a structured
// format, checks for remaining data after unmarshalling, verifies the
// presence of a non-empty <id> field, and sends a cancellation signal to
// the publisher with the associated listener and ID. Returns an error
// message if the envelope lacks a valid <id>.
func (a *A) HandleClose(
	req []byte,
	srv server.I,
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
