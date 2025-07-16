package relay

import (
	"orly.dev/app/relay/publish"
	"orly.dev/interfaces/relay"
	"orly.dev/interfaces/server"
	"orly.dev/interfaces/store"
	"orly.dev/utils/context"
)

func (s *Server) Storage() store.I { return s.relay.Storage() }

func (s *Server) Relay() relay.I { return s.relay }

func (s *Server) Disconnect() { s.disconnect() }

func (s *Server) Publisher() *publish.S { return s.listeners }

func (s *Server) Context() context.T { return s.Ctx }

func (s *Server) AuthRequired() bool { return s.authRequired }

func (s *Server) PublicReadable() bool { return s.publicReadable }

var _ server.I = &Server{}
