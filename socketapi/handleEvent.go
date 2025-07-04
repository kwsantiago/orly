package socketapi

import (
	"bytes"
	"github.com/minio/sha256-simd"
	"not.realy.lol/chk"
	"not.realy.lol/context"
	"not.realy.lol/envelopes/eventenvelope"
	"not.realy.lol/envelopes/okenvelope"
	"not.realy.lol/event"
	"not.realy.lol/filter"
	"not.realy.lol/hex"
	"not.realy.lol/interfaces/server"
	"not.realy.lol/interfaces/store"
	"not.realy.lol/ints"
	"not.realy.lol/kind"
	"not.realy.lol/log"
	"not.realy.lol/tag"
)

func (a *A) HandleEvent(r []byte, s server.I, remote string) (msg []byte) {

	log.T.F("%s handleEvent %s", remote, r)
	var err error
	var ok bool
	var rem []byte
	sto := s.Storage()
	if sto == nil {
		panic("no event store has been set to store event")
	}
	env := eventenvelope.NewSubmission()
	if rem, err = env.Unmarshal(r); chk.E(err) {
		return
	}
	if len(rem) > 0 {
		log.T.F("%s extra '%s'", remote, rem)
	}
	if err = a.VerifyEvent(env); chk.E(err) {
		return
	}
	if env.E.Kind.K == kind.Deletion.K {
		if err = a.CheckDelete(a.Context(), env, sto); chk.E(err) {
			return
		}
	}
	var reason []byte
	ok, reason = s.AddEvent(
		a.Context(), env.E, a.Listener.Req(), remote,
	)
	log.T.F("%s <- event added %v", remote, ok)
	if err = okenvelope.NewFrom(
		env.Id(), ok, reason,
	).Write(a.Listener); chk.E(err) {
		return
	}
	return
}

func (a *A) VerifyEvent(env *eventenvelope.Submission) (err error) {
	if !bytes.Equal(env.GetIDBytes(), env.Id()) {
		if err = Ok.Invalid(
			a, env, "event id is computed incorrectly",
		); chk.E(err) {
			return
		}
		return
	}
	var ok bool
	if ok, err = env.Verify(); chk.T(err) {
		if err = Ok.Error(
			a, env, "failed to verify signature", err,
		); chk.T(err) {
			return
		}
		return
	} else if !ok {
		if err = Ok.Error(a, env, "signature is invalid", err); chk.T(err) {
			return
		}
		return
	}
	return
}

func (a *A) CheckDelete(
	c context.T, env *eventenvelope.Submission, sto store.I,
) (err error) {
	log.I.F("delete event\n%s", env.E.Serialize())
	for _, t := range env.Tags.ToSliceOfTags() {
		var res []*event.E
		if t.Len() >= 2 {
			switch {
			case bytes.Equal(t.Key(), []byte("e")):
				evId := make([]byte, sha256.Size)
				if _, err = hex.DecBytes(evId, t.Value()); chk.E(err) {
					continue
				}
				res, err = sto.QueryEvents(c, &filter.T{IDs: tag.New(evId)})
				if err != nil {
					if err = Ok.Error(
						a, env, "failed to query for target event",
					); chk.T(err) {
						return
					}
					return
				}
				for i := range res {
					if res[i].Kind.Equal(kind.Deletion) {
						if err = Ok.Blocked(
							a, env,
							"not processing or storing delete event containing delete event references",
						); chk.E(err) {
							return
						}
						return
					}
					if !bytes.Equal(res[i].Pubkey, env.E.Pubkey) {
						if err = Ok.Blocked(
							a, env,
							"cannot delete other users' events (delete by e tag)",
						); chk.E(err) {
							return
						}
						return
					}
				}
			case bytes.Equal(t.Key(), []byte("a")):
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
					); chk.T(err) {
					}
					return
				}
				kin := ints.New(uint16(0))
				if _, err = kin.Unmarshal(split[0]); chk.E(err) {
					if err = Ok.Invalid(
						a, env,
						"delete event a tag kind value invalid: %s", t.Value(),
					); chk.T(err) {
						return
					}
					return
				}
				kk := kind.New(kin.Uint16())
				if kk.Equal(kind.Deletion) {
					if err = Ok.Blocked(
						a, env, "delete event kind may not be deleted",
					); chk.E(err) {
						return
					}
					return
				}
				if !kk.IsParameterizedReplaceable() {
					if err = Ok.Error(
						a, env,
						"delete tags with a tags containing non-parameterized-replaceable events cannot be processed",
					); chk.E(err) {
						return
					}
					return
				}
				if !bytes.Equal(pk, env.E.Pubkey) {
					log.I.S(pk, env.E.Pubkey, env.E)
					if err = Ok.Blocked(
						a, env,
						"cannot delete other users' events (delete by a tag)",
					); chk.E(err) {
						return
					}
					return
				}
				f := filter.New()
				f.Kinds.K = []*kind.T{kk}
				f.Authors.Append(pk)
				f.Tags.AppendTags(tag.New([]byte{'#', 'd'}, split[2]))
				if res, err = sto.QueryEvents(c, f); err != nil {
					if err = Ok.Error(
						a, env,
						"failed to query for target event",
					); chk.T(err) {
						return
					}
					return
				}
			}
		}
		if len(res) < 1 {
			continue
		}
		var resTmp event.S
		for _, v := range res {
			if env.E.CreatedAt.U64() >= v.CreatedAt.U64() {
				resTmp = append(resTmp, v)
			}
		}
		res = resTmp
		for _, target := range res {
			var skip bool
			if skip, err = a.ProcessDelete(c, target, env, sto); skip {
				continue
			} else if err != nil {
				return
			}
		}
		res = nil
	}
	if err = okenvelope.NewFrom(env.Id(), true).Write(a.Listener); chk.E(err) {
		return
	}
	return
}

func (a *A) ProcessDelete(
	c context.T, target *event.E, env *eventenvelope.Submission,
	sto store.I,
) (skip bool, err error) {
	if target.Kind.K == kind.Deletion.K {
		if err = Ok.Error(
			a, env, "cannot delete delete event %s", env.Id,
		); chk.E(err) {
			return
		}
	}
	if target.CreatedAt.Int() > env.E.CreatedAt.Int() {
		if err = Ok.Error(
			a, env,
			"not deleting\n%d%\nbecause delete event is older\n%d",
			target.CreatedAt.Int(), env.E.CreatedAt.Int(),
		); chk.E(err) {
			return
		}
		skip = true
	}
	if !bytes.Equal(target.Pubkey, env.Pubkey) {
		if err = Ok.Error(a, env, "only author can delete event"); chk.E(err) {
			return
		}
		return
	}
	if err = sto.DeleteEvent(c, target.EventId()); chk.T(err) {
		if err = Ok.Error(a, env, err.Error()); chk.T(err) {
			return
		}
		return
	}
	return
}
