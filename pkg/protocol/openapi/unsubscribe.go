package openapi

import (
	"github.com/danielgtaylor/huma/v2"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
)

type UnsubscribeInput struct {
	ClientId string `path:"client_id" doc:"Client identifier code associated with subscription channel created with /listen"`
	Id       string `path:"id" doc:"Identifier of the subscription to cancel"`
}

func (x *Operations) RegisterUnsubscribe(api huma.API) {
	name := "Unsubscribe"
	description := `Cancel a subscription with the matching subscription identifier, attached to the identified client_id associated with the HTTP SSE connection.`
	path := x.path + "/unsubscribe/{client_id}/{id}"
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
		}, func(ctx context.T, input *UnsubscribeInput) (
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

			// Check if the client ID exists
			if !CheckListenerExists(input.ClientId, x.Publisher()) {
				return nil, huma.Error404NotFound("client_id doesn't exist, create a listener first with the /listen endpoint")
			}

			// Check if the subscription ID exists
			if !CheckSubscriptionExists(
				input.ClientId, input.Id, x.Publisher(),
			) {
				return nil, huma.Error404NotFound("subscription id doesn't exist for this client")
			}

			// Create a cancel subscription message
			unsubscribe := &H{
				Id: input.ClientId,
				FilterMap: map[string]*filter.F{
					input.Id: nil, // We only need the key, not the value
				},
				Cancel: true, // Set Cancel to true to remove the subscription
			}

			// Send the unsubscribe message to the publisher. The publisher will route it to the appropriate handler based on Type()
			x.Publisher().Receive(unsubscribe)

			log.T.F(
				"removed subscription %s for listener %s", input.Id,
				input.ClientId,
			)

			return &struct{}{}, nil
		},
	)
}
