package socketapi

import (
	"orly.dev/pkg/encoders/envelopes/closeenvelope"
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
)

// HandleClose processes a CLOSE envelope, intended to cancel a specific
// subscription, and notifies the server to handle the cancellation.
//
// Parameters:
//
//   - req: A byte slice containing the raw CLOSE envelope data to process.
//
//   - srv: The server instance responsible for managing subscription
//     operations, such as cancellation.
//
// Return values:
//
//   - note: A byte slice containing an error message if issues occur during
//     processing; otherwise, an empty slice.
//
// Expected behavior:
//
// The method parses and validates the CLOSE envelope. If valid, it cancels the
// corresponding subscription by notifying the server's publisher. If the
// envelope is malformed or the subscription ID is missing, an error message is
// returned instead. Logs any remaining unprocessed data for diagnostics.
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
