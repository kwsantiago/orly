package socketapi

import (
	"bytes"
	"orly.dev/chk"
	"orly.dev/context"
	"orly.dev/envelopes/eventenvelope"
	"orly.dev/envelopes/okenvelope"
	"orly.dev/event"
	"orly.dev/filter"
	"orly.dev/hex"
	"orly.dev/ints"
	"orly.dev/kind"
	"orly.dev/log"
	"orly.dev/normalize"
	"orly.dev/realy/interfaces"
	"orly.dev/tag"
)

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
		if err = okenvelope.NewFrom(
			env.Id, false,
			normalize.Invalid.F("event id is computed incorrectly"),
		).Write(a.Listener); chk.E(err) {
			return
		}
		return
	}
	if ok, err = env.Verify(); chk.T(err) {
		if err = okenvelope.NewFrom(
			env.Id, false,
			normalize.Error.F("failed to verify signature"),
		).Write(a.Listener); chk.E(err) {
			return
		}
	} else if !ok {
		if err = okenvelope.NewFrom(
			env.Id, false,
			normalize.Error.F("signature is invalid"),
		).Write(a.Listener); chk.E(err) {
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
				case bytes.Equal(t.Key(), []byte("a")):
					split := bytes.Split(t.Value(), []byte{':'})
					if len(split) != 3 {
						continue
					}
					// Check if the deletion event is trying to delete itself
					if bytes.Equal(split[2], env.Id) {
						if err = okenvelope.NewFrom(
							env.Id, false,
							normalize.Blocked.F("deletion event cannot reference its own ID"),
						).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					var pk []byte
					if pk, err = hex.DecAppend(nil, split[1]); chk.E(err) {
						if err = okenvelope.NewFrom(
							env.Id, false,
							normalize.Invalid.F(
								"delete event a tag pubkey value invalid: %s",
								t.Value(),
							),
						).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					kin := ints.New(uint16(0))
					if _, err = kin.Unmarshal(split[0]); chk.E(err) {
						if err = okenvelope.NewFrom(
							env.Id, false,
							normalize.Invalid.F(
								"delete event a tag kind value invalid: %s",
								t.Value(),
							),
						).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					kk := kind.New(kin.Uint16())
					if kk.Equal(kind.Deletion) {
						if err = okenvelope.NewFrom(
							env.Id, false,
							normalize.Blocked.F("delete event kind may not be deleted"),
						).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					if !kk.IsParameterizedReplaceable() {
						if err = okenvelope.NewFrom(
							env.Id, false,
							normalize.Error.F("delete tags with a tags containing non-parameterized-replaceable events cannot be processed"),
						).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					if !bytes.Equal(pk, env.E.Pubkey) {
						log.I.S(pk, env.E.Pubkey, env.E)
						if err = okenvelope.NewFrom(
							env.Id, false,
							normalize.Blocked.F("cannot delete other users' events (delete by a tag)"),
						).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					f := filter.New()
					f.Kinds.K = []*kind.T{kk}
					f.Authors.Append(pk)
					f.Tags.AppendTags(tag.New([]byte{'#', 'd'}, split[2]))
					res, err = sto.QueryEvents(c, f)
					if err != nil {
						if err = okenvelope.NewFrom(
							env.Id, false,
							normalize.Error.F("failed to query for target event"),
						).Write(a.Listener); chk.E(err) {
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
					if err = okenvelope.NewFrom(
						env.Id, false,
						normalize.Error.F(
							"cannot delete delete event %s",
							env.Id,
						),
					).Write(a.Listener); chk.E(err) {
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
					if err = okenvelope.NewFrom(
						env.Id, false,
						normalize.Error.F("only author can delete event"),
					).Write(a.Listener); chk.E(err) {
						return
					}
					return
				}
				// Instead of deleting the event, we'll just add the deletion
				// event The query logic will filter out deleted events
				if err = okenvelope.NewFrom(
					env.Id, true,
				).Write(a.Listener); chk.E(err) {
					return
				}
			}
			res = nil
		}
		if err = okenvelope.NewFrom(
			env.Id, true,
		).Write(a.Listener); chk.E(err) {
			return
		}
	}
	var reason []byte
	ok, reason = srv.AddEvent(
		c, rl, env.E, a.Req(), a.RealRemote(), nil,
	)

	log.I.F("event added %v, %s", ok, reason)
	if err = okenvelope.NewFrom(
		env.Id, ok, reason,
	).Write(a.Listener); chk.E(err) {
		return
	}
	return
}
