package publish

import (
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/interfaces/publisher"
	"orly.dev/pkg/interfaces/typer"
)

// S is the control structure for the subscription management scheme.
type S struct {
	publisher.Publishers
}

// New creates a new publish.S.
func New(p ...publisher.I) (s *S) {
	s = &S{Publishers: p}
	return
}

var _ publisher.I = &S{}

func (s *S) Type() string { return "publish" }

func (s *S) Deliver(ev *event.E) {
	for _, p := range s.Publishers {
		p.Deliver(ev)
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
