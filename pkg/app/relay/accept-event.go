package relay

import (
	"net/http"
	"orly.dev/pkg/utils"

	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/utils/context"
)

// AcceptEvent determines whether an incoming event should be accepted for
// processing based on authentication requirements.
//
// # Parameters
//
//   - c: the context of the request
//
//   - ev: pointer to the event structure
//
//   - hr: HTTP request related to the event (if any)
//
//   - authedPubkey: public key of the authenticated user (if any)
//
//   - remote: remote address from where the event was received
//
// # Return Values
//
//   - accept: boolean indicating whether the event should be accepted
//
//   - notice: string providing a message or error notice
//
//   - afterSave: function to execute after saving the event (if applicable)
//
// # Expected Behaviour:
//
// - If authentication is required and no public key is provided, reject the
// event.
//
// - Otherwise, accept the event for processing.
func (s *Server) AcceptEvent(
	c context.T, ev *event.E, hr *http.Request, authedPubkey []byte,
	remote string,
) (accept bool, notice string, afterSave func()) {
	if !s.AuthRequired() {
		// Check blacklist for public relay mode
		if len(s.blacklistPubkeys) > 0 {
			for _, blockedPubkey := range s.blacklistPubkeys {
				if utils.FastEqual(blockedPubkey, ev.Pubkey) {
					notice = "event author is blacklisted"
					return
				}
			}
		}
		accept = true
		return
	}
	// if auth is required and the user is not authed, reject
	if len(authedPubkey) == 0 {
		notice = "client isn't authed"
		return
	}
	for _, u := range s.OwnersMuted() {
		if utils.FastEqual(u, authedPubkey) {
			notice = "event author is banned from this relay"
			return
		}
	}
	// check if the authed user is on the lists
	list := append(s.OwnersFollowed(), s.FollowedFollows()...)
	for _, u := range list {
		if utils.FastEqual(u, authedPubkey) {
			accept = true
			return
		}
	}
	return
}
