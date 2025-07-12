package publisher

import (
	"orly.dev/event"
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
