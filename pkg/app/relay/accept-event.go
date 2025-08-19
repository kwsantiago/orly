package relay

import (
	"net/http"
	"orly.dev/pkg/utils"
	"time"

	"orly.dev/pkg/database"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
)

// AcceptEvent determines whether an incoming event should be accepted for
// processing based on authentication requirements and subscription status.
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
// - If subscriptions are enabled, check subscription status for non-directory events
//
// - If authentication is required and no public key is provided, reject the event.
//
// - Otherwise, accept the event for processing.
func (s *Server) AcceptEvent(
	c context.T, ev *event.E, hr *http.Request, authedPubkey []byte,
	remote string,
) (accept bool, notice string, afterSave func()) {
	// Check subscription if enabled
	if s.C.SubscriptionEnabled {
		// Skip subscription check for directory events (kinds 0, 3, 10002)
		kindInt := ev.Kind.ToInt()
		isDirectoryEvent := kindInt == 0 || kindInt == 3 || kindInt == 10002

		if !isDirectoryEvent {
			// Check cache first
			pubkeyHex := hex.Enc(ev.Pubkey)
			now := time.Now()

			s.subscriptionMutex.RLock()
			cacheExpiry, cached := s.subscriptionCache[pubkeyHex]
			s.subscriptionMutex.RUnlock()

			if cached && now.Before(cacheExpiry) {
				// Cache hit - subscription is active
				accept = true
			} else {
				// Cache miss or expired - check database
				if s.relay != nil && s.relay.Storage() != nil {
					if db, ok := s.relay.Storage().(*database.D); ok {
						isActive, err := db.IsSubscriptionActive(ev.Pubkey)

						if err != nil {
							log.E.F("error checking subscription for %s: %v", pubkeyHex, err)
							notice = "error checking subscription status"
							return
						}

						if !isActive {
							notice = "subscription required - visit relay info page for payment details"
							return
						}

						// Cache positive result for 60 seconds
						s.subscriptionMutex.Lock()
						s.subscriptionCache[pubkeyHex] = now.Add(60 * time.Second)
						s.subscriptionMutex.Unlock()

						accept = true
					} else {
						// Storage is not a database.D, subscription checks disabled
						log.E.F("subscription enabled but storage is not database.D")
					}
				}
			}

			// If subscription check passed, continue with auth checks if needed
			if !accept {
				return
			}
		}
	}

	if !s.AuthRequired() {
		// Check blacklist for public relay mode
		if len(s.blacklistPubkeys) > 0 {
			for _, blockedPubkey := range s.blacklistPubkeys {
				if utils.FastEqual(blockedPubkey, ev.Pubkey) {
					notice = "event author is blacklisted"
					accept = false
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
		accept = false
		return
	}
	for _, u := range s.OwnersMuted() {
		if utils.FastEqual(u, authedPubkey) {
			notice = "event author is banned from this relay"
			accept = false
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
	accept = false
	return
}
