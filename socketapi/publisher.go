package socketapi

import (
	"orly.dev/interfaces/listener"
	"orly.dev/interfaces/typer"
	"orly.dev/log"
	"regexp"
	"sync"

	"orly.dev/chk"
	"orly.dev/envelopes/eventenvelope"
	"orly.dev/event"
	"orly.dev/filters"
	"orly.dev/interfaces/publisher"
	"orly.dev/publish"
)

const Type = "socketapi"

var (
	NIP20prefixmatcher = regexp.MustCompile(`^\w+: `)
)

// Map is a map of filters associated with a collection of ws.Listener connections.
type Map map[listener.I]map[string]*filters.T

type W struct {
	listener.I
	// If Cancel is true, this is a close command.
	Cancel bool
	// Id is the subscription Id. If Cancel is true, cancel the named
	// subscription, otherwise, cancel the publisher for the socket.
	Id       string
	Receiver event.C
	Filters  *filters.T
}

func (w *W) Type() string { return Type }

type Close struct {
	listener.I
	Id string
}

type S struct {
	// Mx is the mutex for the Map.
	Mx sync.Mutex
	// Map is the map of subscribers and subscriptions from the websocket api.
	Map
}

var _ publisher.I = &S{}

func init() {
	publish.P.Register(NewPublisher())
}

func NewPublisher() *S { return &S{Map: make(Map)} }

func (p *S) Type() string { return Type }

func (p *S) Receive(msg typer.T) {
	if m, ok := msg.(*W); ok {
		if m.Cancel {
			if m.Id == "" {
				log.T.F("removing subscriber %s", m.I.Remote())
				p.removeSubscriber(m.I)
			} else {
				log.T.F(
					"removing subscription %s of %s",
					m.Id, m.I.Remote(),
				)
				p.removeSubscriberId(m.I, m.Id)
			}
			return
		}
		p.Mx.Lock()
		if subs, ok := p.Map[m.I]; !ok {
			log.T.F(
				"adding subscription %s for new subscriber %s\n%s", m.Id,
				m.I.Remote(),
				m.Filters.Marshal(nil),
			)
			subs = make(map[string]*filters.T)
			subs[m.Id] = m.Filters
			p.Map[m.I] = subs
		} else {
			log.T.F(
				"adding subscription %s for subscriber %s", m.Id, m.I.Remote(),
			)
			subs[m.Id] = m.Filters
		}
		p.Mx.Unlock()

	}
}

func (p *S) Deliver(ev *event.E) {
	var err error
	// p.Mx.Lock()
	for w, subs := range p.Map {
		for id, subscriber := range subs {
			if !subscriber.Match(ev) {
				continue
			}
			var res *eventenvelope.Result
			if res, err = eventenvelope.NewResultWith(id, ev); chk.E(err) {
				continue
			}
			if err = res.Write(w); chk.E(err) {
				continue
			}
		}
	}
	// p.Mx.Unlock()
}

// removeSubscriberId removes a specific subscription from a subscriber websocket.
func (p *S) removeSubscriberId(ws listener.I, id string) {
	p.Mx.Lock()
	var subs map[string]*filters.T
	var ok bool
	if subs, ok = p.Map[ws]; ok {
		delete(p.Map[ws], id)
		_ = subs
		if len(subs) == 0 {
			delete(p.Map, ws)
		}
	}
	p.Mx.Unlock()
}

// removeSubscriber removes a websocket from the S collection.
func (p *S) removeSubscriber(ws listener.I) {
	p.Mx.Lock()
	clear(p.Map[ws])
	delete(p.Map, ws)
	p.Mx.Unlock()
}
