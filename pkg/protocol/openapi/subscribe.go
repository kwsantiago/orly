package openapi

import (
	"github.com/danielgtaylor/huma/v2"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
)

type SubscribeInput struct {
	Auth     string  `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"false"`
	Accept   string  `header:"Accept" default:"application/nostr+json"`
	ClientId string  `path:"client_id" doc:"Client identifier code associated with subscription channel created with /listen"`
	Id       string  `path:"id" doc:"Identifier of the subscription associated with the filter"`
	Body     *Filter `doc:"filter JSON (standard NIP-01 filter syntax)"`
}

func (x *Operations) RegisterSubscribe(api huma.API) {
	name := "Subscribe"
	description := `Create a new subscription based on a provided filter that will return new events that have arrived matching the subscription filter, over the HTTP SSE channel identified by client_id, created by the /listen endpoint.`
	path := x.path + "/subscribe/{client_id}/{id}"
	scopes := []string{"user", "read"}
	method := http.MethodPost
	huma.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"events"},
			RequestBody: EventsBody,
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		}, func(ctx context.T, input *SubscribeInput) (
			output *struct{}, err error,
		) {
			// Validate client_id exists
			if input.ClientId == "" {
				return nil, huma.Error400BadRequest("client_id is required")
			}

			// Validate subscription ID exists
			if input.Id == "" {
				return nil, huma.Error400BadRequest("subscription id is required")
			}

			// Validate filter exists
			if input.Body == nil {
				return nil, huma.Error400BadRequest("filter is required")
			}

			// Check if the client ID exists
			if !CheckListenerExists(input.ClientId, x.Publisher()) {
				return nil, huma.Error404NotFound("client_id does not exist, create a listener first with the /listen endpoint")
			}

			// Convert the Filter to a filter.F
			f := input.Body.ToFilter()

			// Create a subscription message
			subscription := &H{
				Id: input.ClientId,
				FilterMap: map[string]*filter.F{
					input.Id: f,
				},
			}

			// Send the subscription to the publisher. The publisher will route
			// it to the appropriate handler based on Type()
			x.Publisher().Receive(subscription)

			log.T.F(
				"added subscription %s for listener %s\nfilter %s", input.Id,
				input.ClientId, f.Marshal(nil),
			)

			return
		},
	)
}
