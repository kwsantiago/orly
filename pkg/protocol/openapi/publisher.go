package openapi

import (
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/interfaces/publisher"
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/interfaces/typer"
	"orly.dev/pkg/protocol/auth"
	"orly.dev/pkg/utils/log"
	"reflect"
	"sync"
)

const Type = "openapi"

type Delivery struct {
	SubId string   `json:"sub_id"`
	Event *event.E `json:"event"`
}

type DeliverChan chan *Delivery

type H struct {
	sync.Mutex

	// If Cancel is true, this is a close command (must be done when a listener
	// connection is closed).
	Cancel bool

	// New is a flag that signifies a newly created client_id
	New bool

	// Id is the identifier for an HTTP subscription listener channel
	Id string

	// FilterMap is the collection of filters associated with a listener.
	FilterMap map[string]*filter.F

	// Receiver is the channel for receiving events
	Receiver DeliverChan

	// Pubkey is the authenticated public key for this listener
	Pubkey []byte
}

func (h *H) Type() (typeName string) { return Type }

type Publisher struct {
	sync.Mutex

	// ListenMap maps listener IDs to listener objects
	ListenMap map[string]*H

	// Server is an interface to the server
	Server server.I
}

var _ publisher.I = &Publisher{}

func (p *Publisher) Type() (typeName string) { return Type }

func NewPublisher(s server.I) (p *Publisher) {
	return &Publisher{
		ListenMap: make(map[string]*H),
		Server:    s,
	}
}

// Receive handles incoming messages to manage HTTP listener subscriptions and
// associated filters.
//
// # Parameters
//
// - msg (typer.T): The incoming message to process; expected to be of
// type *H to trigger subscription management actions.
//
// # Expected behaviour
//
// - Checks if the message is of type *H.
//
// - If Cancel is true, removes a subscriber by ID or the entire listener.
//
// - Otherwise, adds the subscription to the map under a mutex lock.
//
// - Logs actions related to subscription creation or removal.
func (p *Publisher) Receive(msg typer.T) {
	if m, ok := msg.(*H); ok {
		if m.Cancel {
			if m.Id == "" {
				// Can't do anything with an empty ID
				log.W.F("received cancel request with empty ID")
				return
			}

			if m.FilterMap == nil || len(m.FilterMap) == 0 {
				// Remove the entire listener
				p.removeListener(m.Id)
				log.T.F("removed listener %s", m.Id)
			} else {
				// Remove specific subscriptions
				p.removeSubscription(m.Id, m.FilterMap)
				for id := range m.FilterMap {
					log.T.F("removed subscription %s for %s", id, m.Id)
				}
			}
			return
		}
		p.Lock()
		defer p.Unlock()
		if listener, ok := p.ListenMap[m.Id]; !ok {
			// Don't create new listeners automatically
			if m.New {
				// Create a new listener when New flag is set
				listener := &H{
					Id:        m.Id,
					FilterMap: make(map[string]*filter.F),
					Receiver:  m.Receiver,
					Pubkey:    m.Pubkey,
				}

				// Add the filters if provided
				if m.FilterMap != nil {
					for id, f := range m.FilterMap {
						listener.FilterMap[id] = f
						log.T.F("added subscription %s for new listener %s", id, m.Id)
					}
				}

				// Add the listener to the map
				p.ListenMap[m.Id] = listener
				log.T.F("added new listener %s", m.Id)
			} else {
				// Only the Listen API should create new listeners
				log.W.F("received message for non-existent listener %s", m.Id)
			}
			return
		} else {
			// Update existing listener
			if m.FilterMap != nil {
				for id, f := range m.FilterMap {
					listener.FilterMap[id] = f
					log.T.F("added subscription %s for %s", id, m.Id)
				}
			}
		}
	}
}

// Deliver processes and distributes an event to all matching subscribers based
// on their filter configurations.
//
// # Parameters
//
// - ev (*event.E): The event to be delivered to subscribed clients.
//
// # Expected behaviour
//
// Delivers the event to all subscribers whose filters match the event. It
// applies authentication checks if required by the server, and skips delivery
// for unauthenticated users when events are privileged.
func (p *Publisher) Deliver(ev *event.E) {
	log.T.F("delivering event %0x to HTTP subscribers", ev.ID)
	p.Lock()
	defer p.Unlock()
	for listenerId, listener := range p.ListenMap {
		for subId, filter := range listener.FilterMap {
			if !filter.Matches(ev) {
				log.I.F(
					"listener %s, subscription id %s event\n%s\ndoes not match filter\n%s",
					listenerId, subId, ev.Marshal(nil),
					filter.Marshal(nil),
				)
				continue
			}
			if p.Server.AuthRequired() {
				if !auth.CheckPrivilege(listener.Pubkey, ev) {
					log.W.F(
						"not privileged %0x ev pubkey %0x listener pubkey %0x kind %s privileged: %v",
						listener.Pubkey, ev.Pubkey,
						listener.Pubkey, ev.Kind.Name(),
						ev.Kind.IsPrivileged(),
					)
					continue
				}
			}
			// Send the event to the listener's receiver channel
			select {
			case listener.Receiver <- &Delivery{SubId: subId, Event: ev}:
				log.T.F(
					"dispatched event %0x to subscription %s for listener %s",
					ev.ID, subId, listenerId,
				)
			default:
				log.W.F(
					"failed to dispatch event %0x to subscription %s for listener %s: channel full",
					ev.ID, subId, listenerId,
				)
			}
		}
	}
}

// removeListener removes a listener from the Publisher collection.
func (p *Publisher) removeListener(id string) {
	p.Lock()
	delete(p.ListenMap, id)
	p.Unlock()
}

// removeSubscription removes specific subscriptions from a listener.
// It does not delete the listener even if all subscriptions are removed.
func (p *Publisher) removeSubscription(
	listenerId string, filterMap map[string]*filter.F,
) {
	p.Lock()
	if listener, ok := p.ListenMap[listenerId]; ok {
		for id := range filterMap {
			delete(listener.FilterMap, id)
		}
		// We no longer delete the listener when all subscriptions are removed
		// This allows the listener to remain active for future subscriptions
	}
	p.Unlock()
}

// ListenerExists checks if a listener with the given ID exists.
func (p *Publisher) ListenerExists(id string) bool {
	p.Lock()
	defer p.Unlock()
	_, exists := p.ListenMap[id]
	return exists
}

// SubscriptionExists checks if a subscription with the given ID exists for a specific listener.
func (p *Publisher) SubscriptionExists(listenerId string, subscriptionId string) bool {
	p.Lock()
	defer p.Unlock()
	listener, exists := p.ListenMap[listenerId]
	if !exists {
		return false
	}
	_, exists = listener.FilterMap[subscriptionId]
	return exists
}

// CheckListenerExists is a package-level function that checks if a listener exists.
// This function is used by the Subscribe and Unsubscribe APIs to check if a client ID exists.
func CheckListenerExists(clientId string, publishers ...publisher.I) bool {
	for _, p := range publishers {
		// Check if the publisher is of type *Publisher
		if pub, ok := p.(*Publisher); ok {
			if pub.ListenerExists(clientId) {
				return true
			}
		}

		// Check if the publisher has a Publishers field of type publisher.Publishers
		// This handles the case where the publisher is a *publish.S
		val := reflect.ValueOf(p)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
			if val.Kind() == reflect.Struct {
				field := val.FieldByName("Publishers")
				if field.IsValid() && field.Type().String() == "publisher.Publishers" {
					// Iterate through the publishers
					for i := 0; i < field.Len(); i++ {
						pub := field.Index(i).Interface().(publisher.I)
						// Check if this publisher is of type *Publisher
						if openPub, ok := pub.(*Publisher); ok {
							if openPub.ListenerExists(clientId) {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

// CheckSubscriptionExists is a package-level function that checks if a subscription exists for a specific listener.
// This function is used by the Unsubscribe API to check if a subscription ID exists before attempting to unsubscribe.
func CheckSubscriptionExists(clientId string, subscriptionId string, publishers ...publisher.I) bool {
	for _, p := range publishers {
		// Check if the publisher is of type *Publisher
		if pub, ok := p.(*Publisher); ok {
			if pub.SubscriptionExists(clientId, subscriptionId) {
				return true
			}
		}

		// Check if the publisher has a Publishers field of type publisher.Publishers
		// This handles the case where the publisher is a *publish.S
		val := reflect.ValueOf(p)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
			if val.Kind() == reflect.Struct {
				field := val.FieldByName("Publishers")
				if field.IsValid() && field.Type().String() == "publisher.Publishers" {
					// Iterate through the publishers
					for i := 0; i < field.Len(); i++ {
						pub := field.Index(i).Interface().(publisher.I)
						// Check if this publisher is of type *Publisher
						if openPub, ok := pub.(*Publisher); ok {
							if openPub.SubscriptionExists(clientId, subscriptionId) {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}
