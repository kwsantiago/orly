package publisher

import (
	"not.realy.lol/event"
	"not.realy.lol/interfaces/typer"
)

type I interface {
	typer.T
	Deliver(ev *event.E)
	Receive(msg typer.T)
}

type Publishers []I
