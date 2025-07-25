package openapi

import (
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"io"
	"net/http"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
	"time"
)

// ImportInput is the parameters of an import operation, authentication and the
// stream of line structured JSON events.
type ImportInput struct {
	Auth string `header:"Authorization" doc:"nostr nip-98 (and expiring variant) token for authentication" required:"true"`
}

// ImportOutput is nothing, basically; a 204 or 200 status is expected.
type ImportOutput struct{}

// RegisterImport is the implementation of the Import operation.
func (x *Operations) RegisterImport(api huma.API) {
	name := "Import"
	description := "Import events from line structured JSON (jsonl)"
	path := x.path + "/import"
	scopes := []string{"admin", "write"}
	method := http.MethodPost
	huma.Register(
		api, huma.Operation{
			OperationID:   name,
			Summary:       name,
			Path:          path,
			Method:        method,
			Tags:          []string{"admin"},
			Description:   helpers.GenerateDescription(description, scopes),
			Security:      []map[string][]string{{"auth": scopes}},
			DefaultStatus: 204,
		}, func(ctx context.T, input *ImportInput) (
			output *ImportOutput, err error,
		) {
			lol.Tracer("Import", input)
			defer func() { lol.Tracer("end Import", output, err) }()
			r := ctx.Value("http-request").(*http.Request)
			remote := helpers.GetRemoteFromReq(r)
			authed, pubkey := x.AdminAuth(r, remote, 10*time.Minute)
			if !authed {
				err = huma.Error401Unauthorized(
					fmt.Sprintf("user %0x not authorized for action", pubkey),
				)
				return
			}
			sto := x.Storage()
			log.I.F(
				"import of event data requested on admin port from %s pubkey %0x",
				remote,
				pubkey,
			)
			read := io.LimitReader(r.Body, r.ContentLength)
			sto.Import(read)
			return
		},
	)
}
