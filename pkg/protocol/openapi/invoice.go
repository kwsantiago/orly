package openapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/encoders/bech32encoding"
	"orly.dev/pkg/protocol/nwc"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/keys"
	"orly.dev/pkg/utils/log"
)

type InvoiceInput struct {
	Auth   string       `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"false"`
	Accept string       `header:"Accept" default:"application/json"`
	Body   *InvoiceBody `doc:"invoice request parameters"`
}

type InvoiceBody struct {
	Pubkey string `json:"pubkey" doc:"user public key in hex or npub format" example:"npub1..."`
	Months int    `json:"months" doc:"number of months subscription (1-12)" minimum:"1" maximum:"12" example:"1"`
}

type InvoiceOutput struct {
	Body *InvoiceResponse
}

type InvoiceResponse struct {
	Bolt11 string `json:"bolt11" doc:"Lightning Network payment request"`
	Amount int64  `json:"amount" doc:"amount in satoshis"`
	Expiry int64  `json:"expiry" doc:"invoice expiration timestamp"`
	Error  string `json:"error,omitempty" doc:"error message if any"`
}

type MakeInvoiceParams struct {
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	Expiry      int64  `json:"expiry,omitempty"`
}

type MakeInvoiceResult struct {
	Bolt11  string `json:"invoice"`
	PayHash string `json:"payment_hash"`
}

// RegisterInvoice implements the POST /api/invoice endpoint for generating Lightning invoices
func (x *Operations) RegisterInvoice(api huma.API) {
	name := "Invoice"
	description := `Generate a Lightning invoice for subscription payment

Creates a Lightning Network invoice for a specified number of months subscription.
The invoice amount is calculated based on the configured monthly price.`
	path := x.path + "/invoice"
	scopes := []string{"user"}
	method := http.MethodPost

	huma.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"payments"},
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		}, func(ctx context.T, input *InvoiceInput) (
			output *InvoiceOutput, err error,
		) {
			output = &InvoiceOutput{Body: &InvoiceResponse{}}

			// Validate input
			if input.Body == nil {
				output.Body.Error = "request body is required"
				return output, huma.Error400BadRequest("request body is required")
			}

			if input.Body.Pubkey == "" {
				output.Body.Error = "pubkey is required"
				return output, huma.Error400BadRequest("pubkey is required")
			}

			if input.Body.Months < 1 || input.Body.Months > 12 {
				output.Body.Error = "months must be between 1 and 12"
				return output, huma.Error400BadRequest("months must be between 1 and 12")
			}

			// Get config from server
			cfg := x.I.Config()
			if cfg.NWCUri == "" {
				output.Body.Error = "NWC not configured"
				return output, huma.Error503ServiceUnavailable("NWC wallet not configured")
			}

			// Validate and convert pubkey format
			var pubkeyBytes []byte
			if pubkeyBytes, err = keys.DecodeNpubOrHex(input.Body.Pubkey); chk.E(err) {
				output.Body.Error = "invalid pubkey format"
				return output, huma.Error400BadRequest("invalid pubkey format: must be hex or npub")
			}

			// Convert to npub for description
			var npub []byte
			if npub, err = bech32encoding.BinToNpub(pubkeyBytes); chk.E(err) {
				output.Body.Error = "failed to convert pubkey to npub"
				log.E.F("failed to convert pubkey to npub: %v", err)
				return output, huma.Error500InternalServerError("failed to process pubkey")
			}

			// Calculate amount based on MonthlyPriceSats config
			totalAmount := cfg.MonthlyPriceSats * int64(input.Body.Months)

			// Create invoice description with npub and month count
			description := fmt.Sprintf("ORLY relay subscription: %d month(s) for %s", input.Body.Months, string(npub))

			// Create NWC client
			var nwcClient *nwc.Client
			if nwcClient, err = nwc.NewClient(cfg.NWCUri); chk.E(err) {
				output.Body.Error = "failed to connect to wallet"
				log.E.F("failed to create NWC client: %v", err)
				return output, huma.Error503ServiceUnavailable("wallet connection failed")
			}

			// Create invoice via NWC make_invoice method
			params := &MakeInvoiceParams{
				Amount:      totalAmount,
				Description: description,
				Expiry:      3600, // 1 hour expiry
			}

			var result MakeInvoiceResult
			if err = nwcClient.Request(ctx, "make_invoice", params, &result); chk.E(err) {
				output.Body.Error = fmt.Sprintf("wallet error: %v", err)
				log.E.F("NWC make_invoice failed: %v", err)
				return output, huma.Error502BadGateway("wallet request failed")
			}

			// Return JSON with bolt11 invoice, amount, and expiry
			output.Body.Bolt11 = result.Bolt11
			output.Body.Amount = totalAmount
			output.Body.Expiry = time.Now().Unix() + 3600 // Current time + 1 hour

			log.I.F("generated invoice for %s: %d sats for %d months", string(npub), totalAmount, input.Body.Months)

			return output, nil
		},
	)
}
