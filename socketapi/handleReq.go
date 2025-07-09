package socketapi

import (
	"orly.dev/chk"
	"orly.dev/envelopes/eoseenvelope"
	"orly.dev/envelopes/eventenvelope"
	"orly.dev/envelopes/reqenvelope"
	"orly.dev/event"
	"orly.dev/interfaces/server"
	"orly.dev/log"
	"orly.dev/publish"
)

func (a *A) HandleReq(
	rem []byte, s server.I, remote string,
) (notice []byte) {
	log.T.F("received request from %s\n%s", remote)
	var err error
	sto := s.Storage()
	if sto == nil {
		panic("no event store has been set to fetch events")
	}
	env := reqenvelope.New()
	if rem, err = env.Unmarshal(rem); chk.E(err) {
		notice = []byte(err.Error())
		return
	}
	log.I.S(env)
	var evs event.S
	for _, f := range env.Filters.F {
		var e event.S
		if e, err = sto.QueryEvents(a.Context(), f); chk.E(err) {
			// this one failed, maybe try another
			err = nil
			continue
		}
		evs = append(evs, e...)
	}
	for _, ev := range evs {
		log.I.F("sending event\n%s", ev.Serialize())
		var res *eventenvelope.Result
		if res, err = eventenvelope.NewResultWith(
			env.Subscription.String(), ev,
		); chk.E(err) {
			return
		}
		if err = res.Write(a.Listener); chk.E(err) {
			return
		}
	}
	if err = eoseenvelope.NewFrom(env.Subscription).Write(a.Listener); chk.E(err) {
		return
	}
	receiver := make(event.C, 32)
	publish.P.Receive(
		&W{
			I:        a.Listener,
			Id:       env.Subscription.String(),
			Receiver: receiver,
			Filters:  env.Filters,
		},
	)

	return
}
