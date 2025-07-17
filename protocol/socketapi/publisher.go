package socketapi

import (
	"orly.dev/interfaces/publisher"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
	"regexp"
	"sync"

	"orly.dev/encoders/envelopes/eventenvelope"
	"orly.dev/encoders/event"
	"orly.dev/encoders/filters"
	"orly.dev/protocol/ws"
)

const Type = "socketapi"

var (
	NIP20prefixmatcher = regexp.MustCompile(`^\w+: `)
)

// Map is a map of filters associated with a collection of ws.Listener
// connections.
type Map map[*ws.Listener]map[string]*filters.T

type W struct {
	*ws.Listener
	// If Cancel is true, this is a close command.
	Cancel bool
	// Id is the subscription Id. If Cancel is true, cancel the named
	// subscription, otherwise, cancel the publisher for the socket.
	Id       string
	Receiver event.C
	Filters  *filters.T
}

func (w *W) Type() (typeName string) { return Type }

type Close struct {
	*ws.Listener
	Id string
}

type S struct {
	// Mx is the mutex for the Map.
	Mx sync.Mutex
	// Map is the map of subscribers and subscriptions from the websocket api.
	Map
}

var _ publisher.I = &S{}

func New() (publisher *S) { return &S{Map: make(Map)} }

func (p *S) Type() (typeName string) { return Type }

func (p *S) Receive(msg publisher.Message) {
	if m, ok := msg.(*W); ok {
		if m.Cancel {
			if m.Id == "" {
				p.removeSubscriber(m.Listener)
				log.T.F("removed listener %s", m.Listener.RealRemote())
			} else {
				p.removeSubscriberId(m.Listener, m.Id)
				log.T.F(
					"removed subscription %s for %s", m.Id,
					m.Listener.RealRemote(),
				)
			}
			return
		}
		p.Mx.Lock()
		if subs, ok := p.Map[m.Listener]; !ok {
			subs = make(map[string]*filters.T)
			subs[m.Id] = m.Filters
			p.Map[m.Listener] = subs
			log.T.F(
				"created new subscription for %s, %s", m.Listener.RealRemote(),
				m.Filters.Marshal(nil),
			)
		} else {
			subs[m.Id] = m.Filters
			log.T.F(
				"added subscription %s for %s", m.Id, m.Listener.RealRemote(),
			)
		}
		p.Mx.Unlock()
	}
}

func (p *S) Deliver(ev *event.E) {
	log.T.F("delivering event %0x to subscribers", ev.Id)
	var err error
	p.Mx.Lock()
	for w, subs := range p.Map {
		log.I.F("%v %s", subs, w.RealRemote())
		for id, subscriber := range subs {
			log.T.F(
				"subscriber %s\n%s", w.RealRemote(),
				subscriber.Marshal(nil),
			)
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
			log.T.F("dispatched event %0x to subscription %s", ev.Id, id)
		}
	}
	p.Mx.Unlock()
}

// removeSubscriberId removes a specific subscription from a subscriber
// websocket.
func (p *S) removeSubscriberId(ws *ws.Listener, id string) {
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
func (p *S) removeSubscriber(ws *ws.Listener) {
	p.Mx.Lock()
	clear(p.Map[ws])
	delete(p.Map, ws)
	p.Mx.Unlock()
}
