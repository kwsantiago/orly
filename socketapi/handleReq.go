package socketapi

import (
	"fmt"
	"orly.dev/chk"
	"orly.dev/envelopes/closedenvelope"
	"orly.dev/envelopes/eoseenvelope"
	"orly.dev/envelopes/eventenvelope"
	"orly.dev/envelopes/reqenvelope"
	"orly.dev/event"
	"orly.dev/interfaces/server"
	"orly.dev/log"
	"orly.dev/publish"
	"sort"
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
	// if the number of events on a filter matches the limit, mark the filter
	// complete to prevent opening a subscription.
	completed := make([]bool, len(env.Filters.F))
	for i, f := range env.Filters.F {
		var e event.S
		if e, err = sto.QueryEvents(a.Context(), f); chk.E(err) {
			// this one failed, maybe try another
			err = nil
			if f.Ids.Len() > 0 {
				completed[i] = true
			}
			continue
		}
		evs = append(evs, e...)
		log.I.S(f.Limit, len(evs), f.Ids.Len())
		if (f.Limit != nil && int(*f.Limit) <= len(evs) && *f.Limit > 0) || f.Ids.Len() > 0 {
			completed[i] = true
		}
	}
	sort.Slice(
		evs, func(i, j int) bool {
			return evs[i].CreatedAt.I64() > evs[j].CreatedAt.I64()
		},
	)
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
	// if all filters are complete, return instead of opening a subscription
	complete := true
	for _, c := range completed {
		if !c {
			complete = false
			break
		}
	}
	if complete {
		log.I.F("all filters complete, returning")
		if err = closedenvelope.NewFrom(
			env.Subscription, []byte(fmt.Sprintf(
				"subscription %s complete", env.Subscription.String(),
			)),
		).Write(a.Listener); chk.E(err) {
			return
		}
		return
	}
	for _, f := range env.Filters.F {
		log.I.F(
			"opening subscription for %s %s", env.Subscription, f.Marshal(nil),
		)
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
