package realy

import (
	"bytes"
	"errors"
	"fmt"
	"orly.dev/encoders/tags"
	"orly.dev/utils/chk"
	"orly.dev/utils/errorf"
	"orly.dev/utils/log"
	"orly.dev/utils/normalize"

	"orly.dev/encoders/event"
	"orly.dev/encoders/filter"
	"orly.dev/encoders/kinds"
	"orly.dev/encoders/tag"
	"orly.dev/interfaces/store"
	"orly.dev/utils/context"
)

// Publish processes and saves an event based on its type and rules.
// It handles replaceable, ephemeral, and parameterized replaceable events.
// Duplicate or conflicting events are managed before saving the new one.
func (s *Server) Publish(c context.T, evt *event.E) (err error) {
	sto := s.relay.Storage()
	if evt.Kind.IsEphemeral() {
		// do not store ephemeral events
		return nil

	} else if evt.Kind.IsReplaceable() {
		// replaceable event, delete before storing
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
					log.I.S(ev, evt)
					return errorf.W(string(normalize.Invalid.F("not replacing newer replaceable event")))
				}
				// not deleting these events because some clients are retarded
				// and the query will pull the new one, but a backup can recover
				// the data of old ones
				if ev.Kind.IsDirectoryEvent() {
					del = false
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
						// replaceable events we don't tombstone when replacing,
						// so if deleted, old versions can be restored
						if err = sto.DeleteEvent(
							c, ev.EventId(), true,
						); chk.E(err) {
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
		f.Tags = tags.New(
			tag.New("#d"), tag.New(evt.Tags.GetFirst(tag.New("d")).Value()),
		)
		log.I.F(
			"filter for parameterized replaceable %v %s",
			f.Tags.ToStringsSlice(),
			f.Serialize(),
		)
		if evs, err = sto.QueryEvents(c, f); err != nil {
			return errorf.E("failed to query before replacing: %w", err)
		}
		log.I.S(evs)
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
						if err = sto.DeleteEvent(
							c, ev.EventId(), true,
						); chk.E(err) {
							return
						}
					}()
				}
			}
		}
	}
	if _, _, err = sto.SaveEvent(c, evt); chk.E(err) && !errors.Is(
		err, store.ErrDupEvent,
	) {
		return
	}
	return
}
