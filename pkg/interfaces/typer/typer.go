// Package typer is an interface for server to use to identify their type simply for
// aggregating multiple self-registered server such that the top level can recognise the
// type of a message and match it to the type of handler.
package typer

type T interface {
	// Type returns a type identifier string to allow multiple self-registering publisher.I to
	// be used with an abstraction to allow multiple APIs to publish.
	Type() string
}
