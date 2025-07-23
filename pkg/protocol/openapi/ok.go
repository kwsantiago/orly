package openapi

import (
	"github.com/danielgtaylor/huma/v2"
	"orly.dev/pkg/encoders/reason"
	"orly.dev/pkg/interfaces/eventId"
)

// OK represents a function that processes events or operations, using provided
// parameters to generate formatted messages and return errors if any issues
// occur during processing.
type OK func(
	a *Operations, env eventId.Ider, format string, params ...any,
) (err error)

// OKs provides a collection of handler functions for managing different types
// of operational outcomes, each corresponding to specific error or status
// conditions such as authentication requirements, rate limiting, and invalid
// inputs.
type OKs struct {
	Ok           OK
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
	Ok: func(
		a *Operations, eid eventId.Ider, format string, params ...any,
	) (err error) {
		return nil
	},
	AuthRequired: func(
		a *Operations, eid eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error401Unauthorized(
			string(
				reason.AuthRequired.F(format, params...),
			),
		)
	},
	PoW: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error406NotAcceptable(
			string(
				reason.PoW.F(format, params...),
			),
		)
	},
	Duplicate: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error422UnprocessableEntity(
			string(
				reason.Duplicate.F(format, params...),
			),
		)
	},
	Blocked: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error406NotAcceptable(
			string(
				reason.Blocked.F(format, params...),
			),
		)
	},
	RateLimited: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error400BadRequest(
			string(
				reason.RateLimited.F(format, params...),
			),
		)
	},
	Invalid: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error422UnprocessableEntity(
			string(
				reason.Invalid.F(format, params...),
			),
		)
	},
	Error: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error500InternalServerError(
			string(
				reason.Error.F(format, params...),
			),
		)
	},
	Unsupported: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error400BadRequest(
			string(
				reason.Unsupported.F(format, params...),
			),
		)
	},
	Restricted: func(
		a *Operations, _ eventId.Ider, format string, params ...any,
	) (err error) {
		return huma.Error403Forbidden(
			string(
				reason.Restricted.F(format, params...),
			),
		)
	},
}
