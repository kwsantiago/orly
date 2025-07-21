package openapi

import (
	"github.com/danielgtaylor/huma/v2"

	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/protocol/servemux"
	"orly.dev/pkg/utils/lol"
)

type Operations struct {
	server.I
	path string
	*servemux.S
}

// New creates a new openapi.Operations and registers its methods.
func New(
	s server.I, name, version, description string, path string,
	sm *servemux.S,
) {
	lol.Tracer("New", name, version, description, path)
	defer func() { lol.Tracer("end New") }()
	a := NewHuma(sm, name, version, description)
	huma.AutoRegister(a, &Operations{I: s, path: path})
	return
}
