package socketapi

import (
	"orly.dev/pkg/encoders/envelopes/okenvelope"
	"orly.dev/pkg/encoders/reason"
	"orly.dev/pkg/interfaces/eventId"
)

// OK represents a function that processes events or operations, using provided
// parameters to generate formatted messages and return errors if any issues
// occur during processing.
type OK func(a *A, env eventId.Ider, format string, params ...any) (err error)

// OKs provides a collection of handler functions for managing different types
// of operational outcomes, each corresponding to specific error or status
// conditions such as authentication requirements, rate limiting, and invalid
// inputs.
type OKs struct {
	AuthRequired OK
	PoW          OK
	Duplicate    OK
	Blocked      OK
	RateLimited  OK
	Invalid      OK
	Error        OK
	Unsupported  OK
	Restricted   OK
}

// Ok provides a collection of handler functions for managing different types of
// operational outcomes, each corresponding to specific error or status
// conditions such as authentication requirements, rate limiting, and invalid
// inputs.
var Ok = OKs{
	AuthRequired: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.AuthRequired.F(format, params...),
		).Write(a.Listener)
	},
	PoW: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.PoW.F(format, params...),
		).Write(a.Listener)
	},
	Duplicate: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.Duplicate.F(format, params...),
		).Write(a.Listener)
	},
	Blocked: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.Blocked.F(format, params...),
		).Write(a.Listener)
	},
	RateLimited: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.RateLimited.F(format, params...),
		).Write(a.Listener)
	},
	Invalid: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.Invalid.F(format, params...),
		).Write(a.Listener)
	},
	Error: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.Error.F(format, params...),
		).Write(a.Listener)
	},
	Unsupported: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.Unsupported.F(format, params...),
		).Write(a.Listener)
	},
	Restricted: func(
		a *A, env eventId.Ider, format string, params ...any,
	) (err error) {
		return okenvelope.NewFrom(
			env.Id(), false, reason.Restricted.F(format, params...),
		).Write(a.Listener)
	},
}
