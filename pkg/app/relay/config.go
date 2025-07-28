package relay

import (
	"orly.dev/pkg/app/config"
)

func (s *Server) Config() (c *config.C) {
	c = s.C
	return
}
