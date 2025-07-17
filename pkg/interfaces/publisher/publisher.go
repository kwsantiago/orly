package publisher

import (
	"orly.dev/pkg/encoders/event"
)

type Message interface {
	Type() string
}

type I interface {
	Message
	Deliver(ev *event.E)
	Receive(msg Message)
}

type Publishers []I
