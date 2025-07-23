package relay

import (
	"bytes"
	"errors"
	"fmt"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/errorf"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/normalize"
)

// Publish processes and stores an event in the server's storage. It handles different types of events: ephemeral, replaceable, and parameterized replaceable.
//
// # Parameters
//
// - c (context.Context): The context for the operation.
//
// - evt (*event.E): The event to be published.
//
// # Return Values
//
// - err (error): An error if any step fails during the publishing process.
//
// # Expected Behaviour
//
// - For ephemeral events, the method doesn't store them and returns
// immediately.
//
// - For replaceable events, it first queries for existing similar events,
// deletes older ones, and then stores the new event.
//
// - For parameterized replaceable events, it performs a similar process but
// uses additional tags to identify duplicates.
func (s *Server) Publish(c context.T, evt *event.E) (err error) {
	sto := s.relay.Storage()
	if evt.Kind.IsEphemeral() {
		// don't store ephemeral events
		return nil

	} else if evt.Kind.IsReplaceable() {
		// replaceable event, delete old after storing
		var evs []*event.E
		f := filter.New()
		f.Authors = tag.New(evt.Pubkey)
		f.Kinds = kinds.New(evt.Kind)
		evs, err = sto.QueryEvents(c, f)
		if err != nil {
			return fmt.Errorf("failed to query before replacing: %w", err)
		}
		if len(evs) > 0 {
			log.T.F("found %d possible duplicate events", len(evs))
			for _, ev := range evs {
				del := true
				if bytes.Equal(ev.Id, evt.Id) {
					continue
				}
				log.I.F(
					"maybe replace %s with %s", ev.Serialize(), evt.Serialize(),
				)
				if ev.CreatedAt.Int() > evt.CreatedAt.Int() {
					return errorf.W(
						string(
							normalize.Invalid.F(
								"not replacing newer replaceable event",
							),
						),
					)
				}
				if evt.Kind.Equal(kind.FollowList) {
					// if the event is from someone on ownersFollowed or
					// followedFollows, for now add to this list so they're
					// immediately effective.
					var isFollowed bool
					ownersFollowed := s.OwnersFollowed()
					for _, pk := range ownersFollowed {
						if bytes.Equal(evt.Pubkey, pk) {
							isFollowed = true
						}
					}
					if isFollowed {
						if _, _, err = sto.SaveEvent(
							c, evt, false,
						); err != nil && !errors.Is(
							err, store.ErrDupEvent,
						) {
							return
						}
						// we need to trigger the spider with no fetch
						if err = s.Spider(true); chk.E(err) {
							err = nil
						}
						// event has been saved and lists updated.
						return
					}

				}
				if evt.Kind.Equal(kind.MuteList) {
					// check if this is one of the owners, if so, the mute list
					// should be applied immediately.
					owners := s.OwnersPubkeys()
					for _, pk := range owners {
						if bytes.Equal(evt.Pubkey, pk) {
							if _, _, err = sto.SaveEvent(
								c, evt, false,
							); err != nil && !errors.Is(
								err, store.ErrDupEvent,
							) {
								return
							}
							// we need to trigger the spider with no fetch
							if err = s.Spider(true); chk.E(err) {
								err = nil
							}
							// event has been saved and lists updated.
							return
						}
					}
				}
				// defer the delete until after the save, further down, has
				// completed.
				if del {
					defer func() {
						if err != nil {
							// something went wrong saving the replacement, so we won't delete
							// the event.
							return
						}
						log.T.C(
							func() string {
								return fmt.Sprintf(
									"%s\nreplacing\n%s", evt.Serialize(),
									ev.Serialize(),
								)
							},
						)
						if err = sto.DeleteEvent(c, ev.EventId()); chk.E(err) {
							return
						}
					}()
				}
			}
		}
	} else if evt.Kind.IsParameterizedReplaceable() {
		log.I.F("parameterized replaceable %s", evt.Serialize())
		// parameterized replaceable event, delete before storing
		var evs []*event.E
		f := filter.New()
		f.Authors = tag.New(evt.Pubkey)
		f.Kinds = kinds.New(evt.Kind)
		// Create a tag with key 'd' and value from the event's d-tag
		dTag := evt.Tags.GetFirst(tag.New("d"))
		if dTag != nil && dTag.Len() > 1 {
			f.Tags = tags.New(
				tag.New([]byte{'d'}, dTag.Value()),
			)
		}
		log.I.F(
			"filter for parameterized replaceable %v %s",
			f.Tags.ToStringsSlice(),
			f.Serialize(),
		)
		if evs, err = sto.QueryEvents(c, f); err != nil {
			return errorf.E("failed to query before replacing: %w", err)
		}
		// log.I.S(evs)
		if len(evs) > 0 {
			for _, ev := range evs {
				del := true
				err = nil
				log.I.F(
					"maybe replace %s with %s", ev.Serialize(), evt.Serialize(),
				)
				if ev.CreatedAt.Int() > evt.CreatedAt.Int() {
					return errorf.D(string(normalize.Error.F("not replacing newer parameterized replaceable event")))
				}
				// not deleting these events because some clients are retarded
				// and the query will pull the new one, but a backup can recover
				// the data of old ones
				if ev.Kind.IsDirectoryEvent() {
					del = false
				}
				evdt := ev.Tags.GetFirst(tag.New("d"))
				evtdt := evt.Tags.GetFirst(tag.New("d"))
				log.I.F(
					"%s != %s %v", evdt.Value(), evtdt.Value(),
					!bytes.Equal(evdt.Value(), evtdt.Value()),
				)
				if !bytes.Equal(evdt.Value(), evtdt.Value()) {
					continue
				}
				if del {
					defer func() {
						if err != nil {
							// something went wrong saving the replacement, so
							// we won't delete the event.
							return
						}
						log.T.C(
							func() string {
								return fmt.Sprintf(
									"%s\nreplacing\n%s", evt.Serialize(),
									ev.Serialize(),
								)
							},
						)
						// replaceable events we don't tombstone when replacing,
						// so if deleted, old versions can be restored
						if err = sto.DeleteEvent(c, ev.EventId()); chk.E(err) {
							return
						}
					}()
				}
			}
		}
	}
	if _, _, err = sto.SaveEvent(c, evt, false); err != nil && !errors.Is(
		err, store.ErrDupEvent,
	) {
		return
	}
	return
}
