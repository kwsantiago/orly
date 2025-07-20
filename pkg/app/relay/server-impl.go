package relay

import (
	"orly.dev/pkg/app/relay/publish"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/interfaces/server"
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/utils/context"
)

func (s *Server) Storage() store.I { return s.relay.Storage() }

func (s *Server) Relay() relay.I { return s.relay }

func (s *Server) Publisher() *publish.S { return s.listeners }

func (s *Server) Context() context.T { return s.Ctx }

func (s *Server) AuthRequired() bool { return s.C.AuthRequired || s.LenOwnersPubkeys() > 0 }

func (s *Server) PublicReadable() bool { return s.C.PublicReadable }

var _ server.I = &Server{}
