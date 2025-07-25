package openapi

import (
	"errors"
	"github.com/danielgtaylor/huma/v2"
	"github.com/dgraph-io/badger/v4"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/protocol/auth"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/pointers"
)

type EventsInput struct {
	Auth   string `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"false"`
	Accept string `header:"Accept" default:"application/nostr+json"`
	Body   []byte `doc:"filter JSON (standard NIP-01 filter syntax)"`
}

type EventsOutput struct {
	Body []byte
}

// RegisterEvents is the implementation of the HTTP API Events method.
//
// This method returns the results of a single filter query, filtered by
// privilege.
func (x *Operations) RegisterEvents(api huma.API) {
	name := "Events"
	description := "query for events, returns raw binary data containing the events in JSON line-structured format (only allows one filter)"
	path := x.path + "/events"
	scopes := []string{"user", "read"}
	method := http.MethodPost
	huma.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"events"},
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		}, func(ctx context.T, input *EventsInput) (
			output *EventsOutput, err error,
		) {
			r := ctx.Value("http-request").(*http.Request)
			remote := helpers.GetRemoteFromReq(r)
			var authed bool
			var pubkey []byte
			// if auth is required and not public readable, the request is not
			// authorized.
			if x.I.AuthRequired() && !x.I.PublicReadable() {
				authed, pubkey = x.UserAuth(r, remote)
				if !authed {
					err = huma.Error401Unauthorized("Not Authorized")
					return
				}
			}
			f := filter.New()
			var rem []byte
			log.I.S(input)
			if len(rem) > 0 {
				log.I.F("extra '%s'", rem)
			}
			var accept bool
			allowed, accept, _ := x.AcceptReq(
				x.Context(), r, filters.New(f), pubkey, remote,
			)
			if !accept {
				err = huma.Error401Unauthorized("Not Authorized for query")
				return
			}
			var events event.S
			for _, ff := range allowed.F {
				// var i uint
				if pointers.Present(ff.Limit) {
					if *ff.Limit == 0 {
						continue
					}
				}
				if events, err = x.Storage().QueryEvents(
					x.Context(), ff,
				); err != nil {
					if errors.Is(err, badger.ErrDBClosed) {
						return
					}
					continue
				}
				// filter events the authed pubkey is not privileged to fetch.
				if x.AuthRequired() && len(pubkey) > 0 {
					var tmp event.S
					for _, ev := range events {
						if !auth.CheckPrivilege(pubkey, ev) {
							log.W.F(
								"not privileged: client pubkey '%0x' event pubkey '%0x' kind %s privileged: %v",
								pubkey, ev.Pubkey,
								ev.Kind.Name(),
								ev.Kind.IsPrivileged(),
							)
							continue
						}
						tmp = append(tmp, ev)
					}
					events = tmp
				}
			}
			for _, ev := range events {
				_ = ev
			}
			return
		},
	)
}
