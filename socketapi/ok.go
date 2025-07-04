package socketapi

import (
	"not.realy.lol/envelopes/eid"
	"not.realy.lol/envelopes/okenvelope"
	"not.realy.lol/reason"
)

type OK func(a *A, env eid.Ider, format string, params ...any) (err error)

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

var Ok = OKs{
	AuthRequired: func(
		a *A, env eid.Ider, format string, params ...any,
	) (err error) {
		rr := reason.AuthRequired.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.AuthRequired.Err(format, params...)
	},
	PoW: func(a *A, env eid.Ider, format string, params ...any) (err error) {
		rr := reason.PoW.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.PoW.Err(format, params...)
	},
	Duplicate: func(
		a *A, env eid.Ider, format string, params ...any,
	) (err error) {
		rr := reason.Duplicate.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.Duplicate.Err(format, params...)
	},
	Blocked: func(
		a *A, env eid.Ider, format string, params ...any,
	) (err error) {
		rr := reason.Blocked.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.Blocked.Err(format, params...)
	},
	RateLimited: func(
		a *A, env eid.Ider, format string, params ...any,
	) (err error) {
		rr := reason.RateLimited.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.RateLimited.Err(format, params...)
	},
	Invalid: func(
		a *A, env eid.Ider, format string, params ...any,
	) (err error) {
		rr := reason.Invalid.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.Invalid.Err(format, params...)
	},
	Error: func(a *A, env eid.Ider, format string, params ...any) (err error) {
		rr := reason.Error.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.Error.Err(format, params...)
	},
	Unsupported: func(
		a *A, env eid.Ider, format string, params ...any,
	) (err error) {
		rr := reason.Unsupported.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.Unsupported.Err(format, params...)
	},
	Restricted: func(
		a *A, env eid.Ider, format string, params ...any,
	) (err error) {
		rr := reason.Restricted.F(format, params...)
		r := okenvelope.NewFrom(
			env.Id(), false, rr,
		)
		r.Write(a.Listener)
		return reason.Restricted.Err(format, params...)
	},
}
