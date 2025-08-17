package socketapi

import (
	"bytes"
	"fmt"
	"orly.dev/pkg/crypto/sha256"
	"orly.dev/pkg/encoders/bech32encoding"
	"orly.dev/pkg/encoders/envelopes/authenvelope"
	"orly.dev/pkg/encoders/envelopes/eventenvelope"
	"orly.dev/pkg/encoders/envelopes/noticeenvelope"
	"orly.dev/pkg/encoders/envelopes/okenvelope"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/eventid"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/ints"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/utils"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/iptracker"
	"orly.dev/pkg/utils/log"
	"strings"
	"time"
)

// HandleEvent processes an incoming event by validating its signature, verifying
// its integrity, and handling deletion operations based on event tags.
//
// # Parameters
//
//   - c (context.T): The context for the current operation, used for logging and
//     cancellation.
//
//   - req ([]byte): The raw byte representation of the event to be processed.
//
//   - srv (server.I): The server interface providing access to storage and relay
//     functionalities required during event handling.
//
// # Return Values
//
//   - msg ([]byte): A byte slice representing a response message, typically empty
//     on success or containing error details if processing fails.
//
// # Expected behaviour
//
// Processes the event by unmarshalling it into an envelope and validating its
// signature. If the event is a deletion, it checks tags to determine which events
// should be deleted, ensuring authorship matches before performing deletions in
// storage. Logs relevant information during processing and returns appropriate
// responses.
func (a *A) HandleEvent(
	c context.T, req []byte, srv server.I,
) (msg []byte) {
	var err error
	log.T.C(
		func() string {
			return fmt.Sprintf(
				"handleEvent %s %s authed: %0x", a.RealRemote(), req,
				a.Listener.AuthedPubkey(),
			)
		},
	)
	var rem []byte
	sto := srv.Storage()
	if sto == nil {
		panic("no event store has been set to store event")
	}
	rl := srv.Relay()
	env := eventenvelope.NewSubmission()
	if rem, err = env.Unmarshal(req); chk.E(err) {
		return
	}
	if len(rem) > 0 {
		log.I.F("extra '%s'", rem)
	}
	if a.I.AuthRequired() && !a.Listener.IsAuthed() {
		remoteIP := a.Listener.RealRemote()
		log.I.F("requesting auth from client from %s", remoteIP)

		// Check if the IP is blocked due to too many failed auth attempts
		if iptracker.Global.IsBlocked(remoteIP) {
			blockedUntil := iptracker.Global.GetBlockedUntil(remoteIP)
			blockMsg := fmt.Sprintf(
				"Too many failed authentication attempts. Blocked until %s",
				blockedUntil.Format(time.RFC3339),
			)

			// Send a notice to the client explaining why they're blocked
			if err = noticeenvelope.NewFrom(blockMsg).Write(a.Listener); chk.E(err) {
				err = nil
			}

			// Close the connection
			log.I.F(
				"closing connection from %s due to too many failed auth attempts",
				remoteIP,
			)
			a.Listener.Close()
			return
		}

		// Record the failed authentication attempt
		blocked := iptracker.Global.RecordFailedAttempt(remoteIP)
		if blocked {
			// If this attempt caused the IP to be blocked, close the connection
			blockedUntil := iptracker.Global.GetBlockedUntil(remoteIP)
			blockMsg := fmt.Sprintf(
				"Too many failed authentication attempts. Blocked until %s",
				blockedUntil.Format(time.RFC3339),
			)

			// Send a notice to the client explaining why they're blocked
			if err = noticeenvelope.NewFrom(blockMsg).Write(a.Listener); chk.E(err) {
				err = nil
			}

			// Close the connection
			log.I.F(
				"closing connection from %s due to too many failed auth attempts",
				remoteIP,
			)
			a.Listener.Close()
			return
		}

		// Continue with normal auth flow for non-blocked IPs
		a.Listener.RequestAuth()
		if err = Ok.AuthRequired(a, env.E, "auth required"); chk.E(err) {
			return
		}
		if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).
			Write(a.Listener); chk.E(err) {
			return
		}
		// send a notice in case the client renders it to explain why auth is required
		opks := a.I.OwnersPubkeys()
		var npubList string
		for i, pk := range opks {
			var npub []byte
			if npub, err = bech32encoding.BinToNpub(pk); chk.E(err) {
				continue
			}
			npubList += string(npub)
			if i < len(opks)-1 {
				npubList += ", "
			}
		}
		if err = noticeenvelope.NewFrom("relay whitelists write access to users within the second degree of the social graph of " + npubList).Write(a.Listener); chk.E(err) {
			err = nil
		}
		// a.Listener.SetPendingEvent(env.E)
		return
	}
	calculatedId := env.E.GetIDBytes()
	if !utils.FastEqual(calculatedId, env.E.ID) {
		if err = Ok.Invalid(
			a, env, "event id is computed incorrectly, "+
				"event has ID %0x, but when computed it is %0x",
			env.E.ID, calculatedId,
		); chk.E(err) {
			return
		}
		return
	}
	if ok, err := env.Verify(); chk.T(err) {
		if err = Ok.Error(
			a, env, fmt.Sprintf(
				"failed to verify signature: %s",
				err.Error(),
			),
		); chk.E(err) {
			return
		}
	} else if !ok {
		if err = Ok.Invalid(
			a, env,
			"signature is invalid",
		); chk.E(err) {
			return
		}
		return
	}
	log.T.F("checking if policy allows this event")
	// check that relay policy allows this event
	accept, notice, _ := srv.AcceptEvent(
		c, env.E, a.Listener.Request, a.Listener.AuthedPubkey(),
		a.Listener.RealRemote(),
	)
	if !accept {
		if strings.Contains(notice, "auth") {
			if err = Ok.AuthRequired(
				a, env, notice,
			); chk.E(err) {
				return
			}

		}
		return
	}
	log.T.F("checking for protected tag")
	// check for protected tag (NIP-70)
	protectedTag := env.E.Tags.GetFirst(tag.New("-"))
	if protectedTag != nil && a.AuthRequired() {
		// check that the pubkey of the event matches the authed pubkey
		if !utils.FastEqual(a.Listener.AuthedPubkey(), env.E.Pubkey) {
			if err = Ok.Blocked(
				a, env,
				"protected tag may only be published by client authed to the same pubkey",
			); chk.E(err) {
				return
			}
		}
	}
	// check and process delete
	if env.E.Kind.K == kind.Deletion.K {
		log.T.C(
			func() string {
				return fmt.Sprintf(
					"delete event\n%s", env.E.Serialize(),
				)
			},
		)
		var ownerDelete bool
		for _, pk := range a.OwnersPubkeys() {
			if utils.FastEqual(pk, env.Pubkey) {
				ownerDelete = true
			}
		}
		for _, t := range env.Tags.ToSliceOfTags() {
			var res []*event.E
			if t.Len() >= 2 {
				switch {
				case utils.FastEqual(t.Key(), []byte("e")):
					// Process 'e' tag (event reference)
					eventId := make([]byte, sha256.Size)
					if _, err = hex.DecBytes(eventId, t.Value()); chk.E(err) {
						return
					}

					// Create a filter to find the referenced event
					f := filter.New()
					f.Ids = f.Ids.Append(eventId)

					// Query for the referenced event
					var referencedEvents []*event.E
					referencedEvents, err = sto.QueryEvents(c, f)
					if chk.E(err) {
						if err = Ok.Error(
							a, env, "failed to query for referenced event",
						); chk.E(err) {
							return
						}
						return
					}

					// If we found the referenced event, check if the author
					// matches
					if len(referencedEvents) > 0 {
						referencedEvent := referencedEvents[0]

						// Check if the author of the deletion event matches the
						// author of the referenced event. Owners can delete
						// anything.
						if !utils.FastEqual(
							referencedEvent.Pubkey, env.Pubkey,
						) && !ownerDelete {
							if err = Ok.Blocked(
								a, env,
								"blocked: can't delete events from other authors",
							); chk.E(err) {
								return
							}
						}

						// Create eventid.T from the event ID bytes
						var eid *eventid.T
						if eid, err = eventid.NewFromBytes(eventId); chk.E(err) {
							if err = Ok.Error(
								a, env, "failed to create event ID",
							); chk.E(err) {
								return
							}
							return
						}

						// Use DeleteEvent to actually delete the referenced
						// event
						if err = sto.DeleteEvent(c, eid); chk.E(err) {
							if err = Ok.Error(
								a, env, "failed to delete referenced event",
							); chk.E(err) {
								return
							}
							return
						}

						log.T.C(
							func() string {
								return fmt.Sprintf(
									"successfully deleted event %x", eventId,
								)
							},
						)
					}
				case utils.FastEqual(t.Key(), []byte("a")):
					split := bytes.Split(t.Value(), []byte{':'})
					if len(split) != 3 {
						continue
					}
					var pk []byte
					if pk, err = hex.DecAppend(nil, split[1]); chk.E(err) {
						if err = Ok.Invalid(
							a, env,
							"delete event a tag pubkey value invalid: %s",
							t.Value(),
						); chk.E(err) {
							return
						}
						return
					}
					kin := ints.New(uint16(0))
					if _, err = kin.Unmarshal(split[0]); chk.E(err) {
						if err = Ok.Invalid(
							a, env, "delete event a tag kind value "+
								"invalid: %s",
							t.Value(),
						); chk.E(err) {
							return
						}
						return
					}
					kk := kind.New(kin.Uint16())
					if kk.Equal(kind.Deletion) {
						if err = Ok.Blocked(
							a, env, "delete event kind may not be "+
								"deleted",
						); chk.E(err) {
							return
						}
						return
					}
					if !kk.IsParameterizedReplaceable() {
						if err = Ok.Error(
							a, env,
							"delete tags with a tags containing "+
								"non-parameterized-replaceable events can't be processed",
						); chk.E(err) {
							return
						}
						return
					}
					if !utils.FastEqual(pk, env.E.Pubkey) && !ownerDelete {
						if err = Ok.Blocked(
							a, env,
							"can't delete other users' events (delete by a tag)",
						); chk.E(err) {
							return
						}
						return
					}
					f := filter.New()
					f.Kinds.K = []*kind.T{kk}
					f.Authors.Append(pk)
					f.Tags.AppendTags(tag.New([]byte{'#', 'd'}, split[2]))
					res, err = sto.QueryEvents(c, f)
					if chk.E(err) {
						if err = Ok.Error(
							a, env, "failed to query for target event",
						); chk.E(err) {
							return
						}
						return
					}
				}
			}
			if len(res) < 1 {
				continue
			}
			var resTmp []*event.E
			for _, v := range res {
				if env.E.CreatedAt.U64() >= v.CreatedAt.U64() {
					resTmp = append(resTmp, v)
				}
			}
			res = resTmp
			for _, target := range res {
				if target.Kind.K == kind.Deletion.K {
					if err = Ok.Error(
						a, env, "cannot delete delete event %s", env.E.ID,
					); chk.E(err) {
						return
					}
				}
				if target.CreatedAt.Int() > env.E.CreatedAt.Int() {
					log.T.C(
						func() string {
							return fmt.Sprintf(
								"not deleting %d because delete event is older %d",
								target.CreatedAt.Int(), env.E.CreatedAt.Int(),
							)
						},
					)
					continue
				}
				if !utils.FastEqual(target.Pubkey, env.Pubkey) && !ownerDelete {
					if err = Ok.Error(
						a, env, "only author can delete event",
					); chk.E(err) {
						return
					}
					return
				}

				// Create eventid.T from the target event ID bytes
				var eid *eventid.T
				eid, err = eventid.NewFromBytes(target.EventId().Bytes())
				if chk.E(err) {
					if err = Ok.Error(
						a, env, "failed to create event ID",
					); chk.E(err) {
						return
					}
					return
				}

				// Use DeleteEvent to actually delete the target event
				if err = sto.DeleteEvent(c, eid); chk.E(err) {
					if err = Ok.Error(
						a, env, "failed to delete target event",
					); chk.E(err) {
						return
					}
					return
				}

				log.T.C(
					func() string {
						return fmt.Sprintf(
							"successfully deleted event %x",
							target.EventId().Bytes(),
						)
					},
				)
				res = nil
			}
		}
		// Send a success response after processing all deletions
		if err = Ok.Ok(a, env, ""); chk.E(err) {
			return
		}
		return
	}
	// Check if this event has been deleted before
	if env.E.Kind.K != kind.Deletion.K {
		// Create a filter to check for deletion events that reference this
		// event ID
		f := filter.New()
		f.Kinds.K = []*kind.T{kind.Deletion}
		f.Tags.AppendTags(tag.New([]byte{'e'}, env.E.ID))

		// Query for deletion events
		var deletionEvents []*event.E
		deletionEvents, err = sto.QueryEvents(c, f)
		if err == nil && len(deletionEvents) > 0 {
			// Found deletion events for this ID, don't save it
			if err = Ok.Blocked(
				a, env, "the event was deleted, not storing it again",
			); chk.E(err) {
				return
			}
			return
		}
	}
	var ok bool
	var reason []byte
	ok, reason = srv.AddEvent(c, rl, env.E, a.Req(), a.RealRemote(), nil)
	log.T.C(
		func() string {
			return fmt.Sprintf(
				"event %0x added %v %s", env.E.ID, ok, reason,
			)
		},
	)
	if err = okenvelope.NewFrom(
		env.E.ID, ok,
	).Write(a.Listener); chk.E(err) {
		return
	}
	return
}
