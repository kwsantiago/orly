package openapi

import (
	"crypto/rand"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
)

type Event struct {
	SubID string   `json:"sub_id"`
	Event *event.J `json:"event"`
}

type ListenInput struct {
	Auth   string `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"false"`
	Accept string `header:"Accept" default:"text/event-stream" enum:"text/event-stream" required:"true"`
}

func (x *Operations) RegisterListen(api huma.API) {
	name := "Listen"
	description := `Opens up a HTTP SSE subscription channel that will send results from the /subscribe endpoint. Writes the subscription channel identifier as the first event in the stream.

Close the connection to end all deliveries, or use /unsubscribe. Before /subscribe or /unsubscribe can be used, this must be first opened.

Many browsers have a limited number of SSE channels that can be open at once, so this allows an app to consolidate all of its subscriptions into one. If the client understands HTTP/2 this limit is relaxed from 6 concurrent subscription channels per domain to 100, the net/http library has supported the protocol transparently since Go 1.6. However, because of the design of this API, each instance of a client only requires one open HTTP SSE connection to receive all subscriptions.`
	path := x.path + "/listen"
	scopes := []string{"user", "read"}
	method := http.MethodPost
	sse.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"events"},
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		},
		map[string]any{
			"client_id": "",
			"event":     &Event{},
		},
		func(ctx context.T, input *ListenInput, send sse.Sender) {
			r := ctx.Value("http-request").(*http.Request)
			remote := helpers.GetRemoteFromReq(r)
			var err error
			var authed bool
			var pubkey []byte
			if x.I.AuthRequired() && !x.I.PublicReadable() {
				authed, pubkey, _ = x.UserAuth(r, remote)
				if !authed {
					err = huma.Error401Unauthorized("Not Authorized")
					return
				}
			}

			// Generate a unique client ID
			id := make([]byte, 16)
			if _, err = rand.Read(id); chk.E(err) {
				return
			}
			clientId := hex.Enc(id)

			// Create a receiver channel for events
			receiver := make(DeliverChan, 32)

			// Create and register the listener
			listener := &H{
				Id:        clientId,
				New:       true,
				Receiver:  receiver,
				Pubkey:    pubkey,
				FilterMap: make(map[string]*filter.F),
			}

			log.T.F("creating new listener %s", clientId)
			x.Publisher().Receive(listener)

			// Send the client ID as the first event
			if err = send.Data(clientId); chk.E(err) {
				return
			}

			// Event loop
		out:
			for {
				select {
				case <-x.Context().Done():
					// server shutdown
					break out
				case <-r.Context().Done():
					// connection has closed
					break out
				case ev := <-receiver:
					// if the channel is closed, the event will be nil
					if ev == nil {
						break out
					}
					if err = send.Data(
						Event{
							clientId, ev.Event.ToEventJ(),
						},
					); chk.E(err) {
						break out
					}
				}
			}
			// Clean up the listener when the context is done
			log.T.F("removing listener %s", clientId)
			listener.Cancel = true
			x.Publisher().Receive(listener)
			return
		},
	)
}
