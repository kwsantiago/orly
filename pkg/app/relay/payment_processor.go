package relay

import (
	"fmt"
	"strings"
	"sync"

	"orly.dev/pkg/app/config"
	"orly.dev/pkg/database"
	"orly.dev/pkg/encoders/bech32encoding"
	"orly.dev/pkg/protocol/nwc"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
)

// PaymentProcessor handles NWC payment notifications and updates subscriptions
type PaymentProcessor struct {
	nwcClient *nwc.Client
	db        *database.D
	config    *config.C
	ctx       context.T
	cancel    context.F
	wg        sync.WaitGroup
}

// NewPaymentProcessor creates a new payment processor
func NewPaymentProcessor(cfg *config.C, db *database.D) (pp *PaymentProcessor, err error) {
	if cfg.NWCUri == "" {
		return nil, fmt.Errorf("NWC URI not configured")
	}

	var nwcClient *nwc.Client
	if nwcClient, err = nwc.NewClient(cfg.NWCUri); chk.E(err) {
		return nil, fmt.Errorf("failed to create NWC client: %w", err)
	}

	ctx, cancel := context.Cancel(context.Bg())

	pp = &PaymentProcessor{
		nwcClient: nwcClient,
		db:        db,
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
	}

	return pp, nil
}

// Start begins listening for payment notifications
func (pp *PaymentProcessor) Start() error {
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		if err := pp.listenForPayments(); err != nil {
			log.E.F("payment processor error: %v", err)
		}
	}()
	return nil
}

// Stop gracefully stops the payment processor
func (pp *PaymentProcessor) Stop() {
	if pp.cancel != nil {
		pp.cancel()
	}
	pp.wg.Wait()
}

// listenForPayments subscribes to NWC notifications and processes payments
func (pp *PaymentProcessor) listenForPayments() error {
	return pp.nwcClient.SubscribeNotifications(pp.ctx, pp.handleNotification)
}

// handleNotification processes incoming payment notifications
func (pp *PaymentProcessor) handleNotification(notificationType string, notification map[string]any) error {
	// Only process payment_received notifications
	if notificationType != "payment_received" {
		return nil
	}

	amount, ok := notification["amount"].(float64)
	if !ok {
		return fmt.Errorf("invalid amount")
	}

	description, _ := notification["description"].(string)
	userNpub := pp.extractNpubFromDescription(description)
	if userNpub == "" {
		if metadata, ok := notification["metadata"].(map[string]any); ok {
			if npubField, ok := metadata["npub"].(string); ok {
				userNpub = npubField
			}
		}
	}
	if userNpub == "" {
		return fmt.Errorf("no npub in payment description")
	}

	pubkey, err := pp.npubToPubkey(userNpub)
	if err != nil {
		return fmt.Errorf("invalid npub: %w", err)
	}

	satsReceived := int64(amount / 1000)
	monthlyPrice := pp.config.MonthlyPriceSats
	if monthlyPrice <= 0 {
		monthlyPrice = 6000
	}

	days := int((float64(satsReceived) / float64(monthlyPrice)) * 30)
	if days < 1 {
		return fmt.Errorf("payment amount too small")
	}

	if err := pp.db.ExtendSubscription(pubkey, days); err != nil {
		return fmt.Errorf("failed to extend subscription: %w", err)
	}

	// Record payment history
	invoice, _ := notification["invoice"].(string)
	preimage, _ := notification["preimage"].(string)
	if err := pp.db.RecordPayment(pubkey, satsReceived, invoice, preimage); err != nil {
		log.E.F("failed to record payment: %v", err)
	}

	log.I.F("payment processed: %s %d sats -> %d days", userNpub, satsReceived, days)

	return nil
}

// extractNpubFromDescription extracts an npub from the payment description
func (pp *PaymentProcessor) extractNpubFromDescription(description string) string {
	// Look for npub1... pattern in the description
	parts := strings.Fields(description)
	for _, part := range parts {
		if strings.HasPrefix(part, "npub1") && len(part) == 63 {
			return part
		}
	}

	// Also check if the entire description is just an npub
	description = strings.TrimSpace(description)
	if strings.HasPrefix(description, "npub1") && len(description) == 63 {
		return description
	}

	return ""
}

// npubToPubkey converts an npub string to pubkey bytes
func (pp *PaymentProcessor) npubToPubkey(npubStr string) ([]byte, error) {
	// Validate npub format
	if !strings.HasPrefix(npubStr, "npub1") || len(npubStr) != 63 {
		return nil, fmt.Errorf("invalid npub format")
	}

	// Decode using bech32encoding
	prefix, value, err := bech32encoding.Decode([]byte(npubStr))
	if err != nil {
		return nil, fmt.Errorf("failed to decode npub: %w", err)
	}

	if !strings.EqualFold(string(prefix), "npub") {
		return nil, fmt.Errorf("invalid prefix: %s", string(prefix))
	}

	pubkey, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("decoded value is not []byte")
	}

	return pubkey, nil
}
