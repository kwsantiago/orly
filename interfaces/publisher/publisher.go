package publisher

import (
	"orly.dev/event"
	"orly.dev/interfaces/typer"
)

type I interface {
	typer.T
	Deliver(ev *event.E)
	Receive(msg typer.T)
}

type Publishers []I
