package openapi

import (
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"orly.dev/pkg/app/relay/helpers"
	"orly.dev/pkg/database"
	"orly.dev/pkg/encoders/bech32encoding"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
)

// SubscriptionInput defines the input for the subscription status endpoint
type SubscriptionInput struct {
	Auth   string `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"false"`
	Pubkey string `path:"pubkey" doc:"User's public key in hex or npub format" maxLength:"64" minLength:"52"`
}

// SubscriptionOutput defines the response for the subscription status endpoint
type SubscriptionOutput struct {
	Body SubscriptionStatus `json:"subscription"`
}

// SubscriptionStatus contains the subscription information for a user
type SubscriptionStatus struct {
	TrialEnd      *time.Time `json:"trial_end,omitempty"`
	PaidUntil     *time.Time `json:"paid_until,omitempty"`
	IsActive      bool       `json:"is_active"`
	DaysRemaining *int       `json:"days_remaining,omitempty"`
}

// parsePubkey converts either hex or npub format pubkey to bytes
func parsePubkey(pubkeyStr string) (pubkey []byte, err error) {
	pubkeyStr = strings.TrimSpace(pubkeyStr)

	// Check if it's npub format
	if strings.HasPrefix(pubkeyStr, "npub") {
		if pubkey, err = bech32encoding.NpubToBytes([]byte(pubkeyStr)); err != nil {
			return nil, err
		}
		return pubkey, nil
	}

	// Assume it's hex format
	if pubkey, err = hex.DecodeString(pubkeyStr); err != nil {
		return nil, err
	}

	// Validate length (should be 32 bytes for a public key)
	if len(pubkey) != 32 {
		err = log.E.Err("invalid pubkey length: expected 32 bytes, got %d", len(pubkey))
		return nil, err
	}

	return pubkey, nil
}

// calculateDaysRemaining calculates the number of days remaining in the subscription
func calculateDaysRemaining(sub *database.Subscription) *int {
	if sub == nil {
		return nil
	}

	now := time.Now()
	var activeUntil time.Time

	// Check if trial is active
	if now.Before(sub.TrialEnd) {
		activeUntil = sub.TrialEnd
	} else if !sub.PaidUntil.IsZero() && now.Before(sub.PaidUntil) {
		activeUntil = sub.PaidUntil
	} else {
		// No active subscription
		return nil
	}

	days := int(activeUntil.Sub(now).Hours() / 24)
	if days < 0 {
		days = 0
	}

	return &days
}

// RegisterSubscription implements the subscription status API endpoint
func (x *Operations) RegisterSubscription(api huma.API) {
	name := "Subscription"
	description := `Get subscription status for a user by their public key

Returns subscription information including trial status, paid subscription status, 
active status, and days remaining.`
	path := x.path + "/subscription/{pubkey}"
	scopes := []string{"user", "read"}
	method := http.MethodGet

	huma.Register(
		api, huma.Operation{
			OperationID: name,
			Summary:     name,
			Path:        path,
			Method:      method,
			Tags:        []string{"subscription"},
			Description: helpers.GenerateDescription(description, scopes),
			Security:    []map[string][]string{{"auth": scopes}},
		}, func(ctx context.T, input *SubscriptionInput) (
			output *SubscriptionOutput, err error,
		) {
			r := ctx.Value("http-request").(*http.Request)
			remote := helpers.GetRemoteFromReq(r)

			// Rate limiting check - simple in-memory rate limiter
			// TODO: Implement proper distributed rate limiting

			// Parse pubkey from either hex or npub format
			var pubkey []byte
			if pubkey, err = parsePubkey(input.Pubkey); err != nil {
				err = huma.Error400BadRequest("Invalid pubkey format", err)
				return
			}

			// Get subscription manager
			storage := x.Storage()
			db, ok := storage.(*database.D)
			if !ok {
				err = huma.Error500InternalServerError("Database error")
				return
			}

			var sub *database.Subscription
			if sub, err = db.GetSubscription(pubkey); err != nil {
				err = huma.Error500InternalServerError("Failed to retrieve subscription", err)
				return
			}

			// Handle non-existent subscriptions gracefully
			var status SubscriptionStatus
			if sub == nil {
				// No subscription exists yet
				status = SubscriptionStatus{
					IsActive:      false,
					DaysRemaining: nil,
				}
			} else {
				now := time.Now()
				isActive := false

				// Check if trial is active or paid subscription is active
				if now.Before(sub.TrialEnd) || (!sub.PaidUntil.IsZero() && now.Before(sub.PaidUntil)) {
					isActive = true
				}

				status = SubscriptionStatus{
					IsActive:      isActive,
					DaysRemaining: calculateDaysRemaining(sub),
				}

				// Include trial_end if it's set and in the future
				if !sub.TrialEnd.IsZero() {
					status.TrialEnd = &sub.TrialEnd
				}

				// Include paid_until if it's set
				if !sub.PaidUntil.IsZero() {
					status.PaidUntil = &sub.PaidUntil
				}
			}

			log.I.F("subscription status request for pubkey %x from %s: active=%v, days_remaining=%v",
				pubkey, remote, status.IsActive, status.DaysRemaining)

			output = &SubscriptionOutput{
				Body: status,
			}
			return
		},
	)
}
