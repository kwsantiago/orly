package publisher

import (
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/interfaces/typer"
)

type I interface {
	typer.T
	Deliver(ev *event.E)
	Receive(msg typer.T)
}

type Publishers []I
