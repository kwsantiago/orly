// Package relay contains a collection of server for enabling the building
// of modular nostr relay implementations.
package relay

import (
	"orly.dev/pkg/interfaces/store"
	"orly.dev/pkg/protocol/relayinfo"
	"orly.dev/pkg/utils/context"
)

// I is the main interface for implementing a nostr relay.
type I interface {
	// Name is used as the "name" field in NIP-11 and as a prefix in default
	// S logging. For other NIP-11 fields, see [Informer].
	Name() string
	// Init is called at the very beginning by [S.Start], allowing a realy
	// to initialize its internal resources.
	Init() error
	// Storage returns the relay storage implementation.
	Storage() store.I
}

// Informer is called to compose NIP-11 response to an HTTP request
// with application/nostr+json mime type.
// See also [I.Name].
type Informer interface {
	GetNIP11InformationDocument() *relayinfo.T
}

// ShutdownAware is called during the server shutdown.
// See [S.Shutdown] for details.
type ShutdownAware interface {
	OnShutdown(context.T)
}

// Logger is what [S] uses to log messages.
type Logger interface {
	Infof(format string, v ...any)
	Warningf(format string, v ...any)
	Errorf(format string, v ...any)
}
