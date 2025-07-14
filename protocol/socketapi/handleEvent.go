package socketapi

import (
	"bytes"
	"orly.dev/app/realy/interfaces"
	"orly.dev/encoders/envelopes/eventenvelope"
	"orly.dev/encoders/envelopes/okenvelope"
	"orly.dev/encoders/event"
	"orly.dev/encoders/eventid"
	"orly.dev/encoders/filter"
	"orly.dev/encoders/hex"
	"orly.dev/encoders/ints"
	"orly.dev/encoders/kind"
	"orly.dev/encoders/tag"
	"orly.dev/utils/chk"
	"orly.dev/utils/context"
	"orly.dev/utils/log"
	"orly.dev/utils/normalize"
)

// sendResponse is a helper function to send an okenvelope response and handle errors
func (a *A) sendResponse(eventID []byte, ok bool, reason ...[]byte) error {
	var r []byte
	if len(reason) > 0 {
		r = reason[0]
	}
	return okenvelope.NewFrom(eventID, ok, r).Write(a.Listener)
}

func (a *A) HandleEvent(
	c context.T, req []byte, srv interfaces.Server,
) (msg []byte) {

	log.T.F("handleEvent %s %s", a.RealRemote(), req)
	var err error
	var ok bool
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
	if !bytes.Equal(env.GetIDBytes(), env.Id) {
		if err = a.sendResponse(
			env.Id, false,
			normalize.Invalid.F("event id is computed incorrectly"),
		); chk.E(err) {
			return
		}
		return
	}
	if ok, err = env.Verify(); chk.T(err) {
		if err = a.sendResponse(
			env.Id, false, normalize.Error.F("failed to verify signature"),
		); chk.E(err) {
			return
		}
	} else if !ok {
		if err = a.sendResponse(
			env.Id, false, normalize.Error.F("signature is invalid"),
		); chk.E(err) {
			return
		}
		return
	}
	if env.E.Kind.K == kind.Deletion.K {
		log.I.F("delete event\n%s", env.E.Serialize())
		for _, t := range env.Tags.ToSliceOfTags() {
			var res []*event.E
			if t.Len() >= 2 {
				switch {
				case bytes.Equal(t.Key(), []byte("e")):
					// Process 'e' tag (event reference)
					eventID := t.Value()

					// Create a filter to find the referenced event
					f := filter.New()
					f.Ids = f.Ids.Append(eventID)

					// Query for the referenced event
					var referencedEvents []*event.E
					referencedEvents, err = sto.QueryEvents(c, f)
					if chk.E(err) {
						if err = a.sendResponse(
							env.Id, false,
							normalize.Error.F("failed to query for referenced event"),
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
						// author of the referenced event
						if !bytes.Equal(referencedEvent.Pubkey, env.Pubkey) {
							if err = a.sendResponse(
								env.Id, false,
								normalize.Blocked.F("blocked: cannot delete events from other authors"),
							); chk.E(err) {
								return
							}
							return
						}

						// Create eventid.T from the event ID bytes
						var eid *eventid.T
						eid, err = eventid.NewFromBytes(eventID)
						if chk.E(err) {
							if err = a.sendResponse(
								env.Id, false,
								normalize.Error.F("failed to create event ID"),
							); chk.E(err) {
								return
							}
							return
						}

						// Use DeleteEvent to actually delete the referenced
						// event
						if err = sto.DeleteEvent(c, eid); chk.E(err) {
							if err = a.sendResponse(
								env.Id, false,
								normalize.Error.F("failed to delete referenced event"),
							); chk.E(err) {
								return
							}
							return
						}

						log.I.F("successfully deleted event %x", eventID)
					}
				case bytes.Equal(t.Key(), []byte("a")):
					split := bytes.Split(t.Value(), []byte{':'})
					if len(split) != 3 {
						continue
					}
					// Check if the deletion event is trying to delete itself
					if bytes.Equal(split[2], env.Id) {
						if err = a.sendResponse(
							env.Id, false,
							normalize.Blocked.F("deletion event cannot reference its own ID"),
						); chk.E(err) {
							return
						}
						return
					}
					var pk []byte
					if pk, err = hex.DecAppend(nil, split[1]); chk.E(err) {
						if err = a.sendResponse(
							env.Id, false, normalize.Invalid.F(
								"delete event a tag pubkey value invalid: %s",
								t.Value(),
							),
						); chk.E(err) {
							return
						}
						return
					}
					kin := ints.New(uint16(0))
					if _, err = kin.Unmarshal(split[0]); chk.E(err) {
						if err = a.sendResponse(
							env.Id, false, normalize.Invalid.F(
								"delete event a tag kind value invalid: %s",
								t.Value(),
							),
						); chk.E(err) {
							return
						}
						return
					}
					kk := kind.New(kin.Uint16())
					if kk.Equal(kind.Deletion) {
						if err = a.sendResponse(
							env.Id, false,
							normalize.Blocked.F("delete event kind may not be deleted"),
						); chk.E(err) {
							return
						}
						return
					}
					if !kk.IsParameterizedReplaceable() {
						if err = a.sendResponse(
							env.Id, false,
							normalize.Error.F("delete tags with a tags containing non-parameterized-replaceable events cannot be processed"),
						); chk.E(err) {
							return
						}
						return
					}
					if !bytes.Equal(pk, env.E.Pubkey) {
						log.I.S(pk, env.E.Pubkey, env.E)
						if err = a.sendResponse(
							env.Id, false,
							normalize.Blocked.F("cannot delete other users' events (delete by a tag)"),
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
						if err = a.sendResponse(
							env.Id, false,
							normalize.Error.F("failed to query for target event"),
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
					if err = a.sendResponse(
						env.Id, false, normalize.Error.F(
							"cannot delete delete event %s", env.Id,
						),
					); chk.E(err) {
						return
					}
				}
				if target.CreatedAt.Int() > env.E.CreatedAt.Int() {
					log.I.F(
						"not deleting\n%d%\nbecause delete event is older\n%d",
						target.CreatedAt.Int(), env.E.CreatedAt.Int(),
					)
					continue
				}
				if !bytes.Equal(target.Pubkey, env.Pubkey) {
					if err = a.sendResponse(
						env.Id, false,
						normalize.Error.F("only author can delete event"),
					); chk.E(err) {
						return
					}
					return
				}

				// Create eventid.T from the target event ID bytes
				var eid *eventid.T
				eid, err = eventid.NewFromBytes(target.EventId().Bytes())
				if chk.E(err) {
					if err = a.sendResponse(
						env.Id, false,
						normalize.Error.F("failed to create event ID"),
					); chk.E(err) {
						return
					}
					return
				}

				// Use DeleteEvent to actually delete the target event
				// with noTombstone=true to not save tombstones
				if err = sto.DeleteEvent(c, eid, true); chk.E(err) {
					if err = a.sendResponse(
						env.Id, false,
						normalize.Error.F("failed to delete target event"),
					); chk.E(err) {
						return
					}
					return
				}

				log.I.F("successfully deleted event %x", target.EventId().Bytes())
			}
			res = nil
		}
		// Send success response after processing all deletions
		if err = a.sendResponse(env.Id, true); chk.E(err) {
			return
		}
		// Check if this event has been deleted before
		if env.E.Kind.K != kind.Deletion.K {
			// Create a filter to check for deletion events that reference this
			// event ID
			f := filter.New()
			f.Kinds.K = []*kind.T{kind.Deletion}
			f.Tags.AppendTags(tag.New([]byte{'e'}, env.Id))

			// Query for deletion events
			var deletionEvents []*event.E
			deletionEvents, err = sto.QueryEvents(c, f)
			if err == nil && len(deletionEvents) > 0 {
				// Found deletion events for this ID, don't save it
				if err = a.sendResponse(
					env.Id, false,
					normalize.Blocked.F("event was deleted, not storing it again"),
				); chk.E(err) {
					return
				}
				return
			}
		}
	}
	var reason []byte
	ok, reason = srv.AddEvent(
		c, rl, env.E, a.Req(), a.RealRemote(), nil,
	)

	log.I.F("event added %v, %s", ok, reason)
	if err = a.sendResponse(env.Id, ok, reason); chk.E(err) {
		return
	}
	return
}
