// Package publisher is a singleton package that keeps track of subscriptions
// from relevant API connections.
package publish

import (
	"orly.dev/event"
	"orly.dev/interfaces/publisher"
	"orly.dev/interfaces/typer"
)

var P = &S{}

func (s *S) Register(p publisher.I) {
	s.Publishers = append(s.Publishers, p)
}

// S is the control structure for the subscription management scheme.
type S struct {
	publisher.Publishers
}

// New creates a new publisher.
func New(p ...publisher.I) (s *S) {
	s = &S{Publishers: p}
	return
}

func (s *S) Type() string { return "publish" }

func (s *S) Deliver(ev *event.E) {
	for _, p := range s.Publishers {
		p.Deliver(ev)
		return
	}
}

func (s *S) Receive(msg typer.T) {
	t := msg.Type()
	for _, p := range s.Publishers {
		if p.Type() == t {
			p.Receive(msg)
			return
		}
	}
}
