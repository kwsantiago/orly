package openapi

import (
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
)

// ExportInput is the parameters for the HTTP API Export method.
type ExportInput struct {
	Auth string `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"true"`
}

// ExportOutput is the return value of Export. It usually will be line
// structured JSON.
type ExportOutput struct{ RawBody []byte }

// RegisterExport implements the Export HTTP API method.
func (x *Operations) RegisterExport(api huma.API) {
	name := "Export"
	description := `Export all events (only works with NIP-98 capable client, will not work with UI)

Returns the events as line structured JSON (JSONL) in the order that they were received by the relay.`
	path := x.path + "/export"
	scopes := []string{"admin", "read"}
	method := http.MethodGet
	huma.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"admin"},
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		}, func(ctx context.T, input *ExportInput) (
			resp *huma.StreamResponse, err error,
		) {
			r := ctx.Value("http-request").(*http.Request)
			remote := helpers.GetRemoteFromReq(r)
			log.I.F("processing export from %s", remote)
			authed, pubkey := x.AdminAuth(r, remote)
			if !authed {
				err = huma.Error401Unauthorized("Not Authorized")
				return
			}
			log.T.C(
				func() string {
					return fmt.Sprintf(
						"%s export of event data requested on admin port pubkey %0x",
						remote, pubkey,
					)
				},
			)
			sto := x.Storage()
			resp = &huma.StreamResponse{
				Body: func(ctx huma.Context) {
					ctx.SetHeader("Content-Type", "application/nostr+jsonl")
					sto.Export(x.Context(), ctx.BodyWriter())
					if f, ok := ctx.BodyWriter().(http.Flusher); ok {
						f.Flush()
					} else {
						log.W.F("error: unable to flush")
					}
				},
			}
			return
		},
	)
}
