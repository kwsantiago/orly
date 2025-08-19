package nwc

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"orly.dev/pkg/crypto/encryption"
	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
)

// MockWalletService implements a mock NIP-47 wallet service for testing
type MockWalletService struct {
	relay            string
	walletSecretKey  signer.I
	walletPublicKey  []byte
	client           *ws.Client
	ctx              context.T
	cancel           context.F
	balance          int64 // in satoshis
	balanceMutex     sync.RWMutex
	connectedClients map[string][]byte // pubkey -> conversation key
	clientsMutex     sync.RWMutex
}

// NewMockWalletService creates a new mock wallet service
func NewMockWalletService(relay string, initialBalance int64) (service *MockWalletService, err error) {
	// Generate wallet keypair
	walletKey := &p256k.Signer{}
	if err = walletKey.Generate(); chk.E(err) {
		return
	}

	ctx, cancel := context.Cancel(context.Bg())

	service = &MockWalletService{
		relay:            relay,
		walletSecretKey:  walletKey,
		walletPublicKey:  walletKey.Pub(),
		ctx:              ctx,
		cancel:           cancel,
		balance:          initialBalance,
		connectedClients: make(map[string][]byte),
	}
	return
}

// Start begins the mock wallet service
func (m *MockWalletService) Start() (err error) {
	// Connect to relay
	if m.client, err = ws.RelayConnect(m.ctx, m.relay); chk.E(err) {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	// Publish wallet info event
	if err = m.publishWalletInfo(); chk.E(err) {
		return fmt.Errorf("failed to publish wallet info: %w", err)
	}

	// Subscribe to request events
	if err = m.subscribeToRequests(); chk.E(err) {
		return fmt.Errorf("failed to subscribe to requests: %w", err)
	}

	return
}

// Stop stops the mock wallet service
func (m *MockWalletService) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.client != nil {
		m.client.Close()
	}
}

// GetWalletPublicKey returns the wallet's public key
func (m *MockWalletService) GetWalletPublicKey() []byte {
	return m.walletPublicKey
}

// publishWalletInfo publishes the NIP-47 info event (kind 13194)
func (m *MockWalletService) publishWalletInfo() (err error) {
	capabilities := []string{
		"get_info",
		"get_balance",
		"make_invoice",
		"pay_invoice",
	}

	info := map[string]any{
		"capabilities":  capabilities,
		"notifications": []string{"payment_received", "payment_sent"},
	}

	var content []byte
	if content, err = json.Marshal(info); chk.E(err) {
		return
	}

	ev := &event.E{
		Content:   content,
		CreatedAt: timestamp.Now(),
		Kind:      kind.New(13194),
		Tags:      tags.New(),
	}

	if err = ev.Sign(m.walletSecretKey); chk.E(err) {
		return
	}

	return m.client.Publish(m.ctx, ev)
}

// subscribeToRequests subscribes to NWC request events (kind 23194)
func (m *MockWalletService) subscribeToRequests() (err error) {
	var sub *ws.Subscription
	if sub, err = m.client.Subscribe(
		m.ctx, filters.New(
			&filter.F{
				Kinds: kinds.New(kind.New(23194)),
				Tags: tags.New(
					tag.New("p", hex.Enc(m.walletPublicKey)),
				),
				Since: &timestamp.T{V: time.Now().Unix()},
			},
		),
	); chk.E(err) {
		return
	}

	// Handle incoming request events
	go m.handleRequestEvents(sub)
	return
}

// handleRequestEvents processes incoming NWC request events
func (m *MockWalletService) handleRequestEvents(sub *ws.Subscription) {
	for {
		select {
		case <-m.ctx.Done():
			return
		case ev := <-sub.Events:
			if ev == nil {
				continue
			}
			if err := m.processRequestEvent(ev); chk.E(err) {
				fmt.Printf("Error processing request event: %v\n", err)
			}
		}
	}
}

// processRequestEvent processes a single NWC request event
func (m *MockWalletService) processRequestEvent(ev *event.E) (err error) {
	// Get client pubkey from event
	clientPubkey := ev.Pubkey
	clientPubkeyHex := hex.Enc(clientPubkey)

	// Generate or get conversation key
	var conversationKey []byte
	m.clientsMutex.Lock()
	if existingKey, exists := m.connectedClients[clientPubkeyHex]; exists {
		conversationKey = existingKey
	} else {
		if conversationKey, err = encryption.GenerateConversationKeyWithSigner(
			m.walletSecretKey, clientPubkey,
		); chk.E(err) {
			m.clientsMutex.Unlock()
			return
		}
		m.connectedClients[clientPubkeyHex] = conversationKey
	}
	m.clientsMutex.Unlock()

	// Decrypt request content
	var decrypted []byte
	if decrypted, err = encryption.Decrypt(ev.Content, conversationKey); chk.E(err) {
		return
	}

	var request map[string]any
	if err = json.Unmarshal(decrypted, &request); chk.E(err) {
		return
	}

	method, ok := request["method"].(string)
	if !ok {
		return fmt.Errorf("invalid method")
	}

	params := request["params"]

	// Process the method
	var result any
	if result, err = m.processMethod(method, params); chk.E(err) {
		// Send error response
		return m.sendErrorResponse(clientPubkey, conversationKey, "INTERNAL", err.Error())
	}

	// Send success response
	return m.sendSuccessResponse(clientPubkey, conversationKey, result)
}

// processMethod handles the actual NWC method execution
func (m *MockWalletService) processMethod(method string, params any) (result any, err error) {
	switch method {
	case "get_info":
		return m.getInfo()
	case "get_balance":
		return m.getBalance()
	case "make_invoice":
		return m.makeInvoice(params)
	case "pay_invoice":
		return m.payInvoice(params)
	default:
		err = fmt.Errorf("unsupported method: %s", method)
		return
	}
}

// getInfo returns wallet information
func (m *MockWalletService) getInfo() (result map[string]any, err error) {
	result = map[string]any{
		"alias":        "Mock Wallet",
		"color":        "#3399FF",
		"pubkey":       hex.Enc(m.walletPublicKey),
		"network":      "mainnet",
		"block_height": 850000,
		"block_hash":   "0000000000000000000123456789abcdef",
		"methods":      []string{"get_info", "get_balance", "make_invoice", "pay_invoice"},
	}
	return
}

// getBalance returns the current wallet balance
func (m *MockWalletService) getBalance() (result map[string]any, err error) {
	m.balanceMutex.RLock()
	balance := m.balance
	m.balanceMutex.RUnlock()

	result = map[string]any{
		"balance": balance * 1000, // convert to msats
	}
	return
}

// makeInvoice creates a Lightning invoice
func (m *MockWalletService) makeInvoice(params any) (result map[string]any, err error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		err = fmt.Errorf("invalid params")
		return
	}

	amount, ok := paramsMap["amount"].(float64)
	if !ok {
		err = fmt.Errorf("missing or invalid amount")
		return
	}

	description := ""
	if desc, ok := paramsMap["description"].(string); ok {
		description = desc
	}

	paymentHash := make([]byte, 32)
	rand.Read(paymentHash)

	// Generate a fake bolt11 invoice
	bolt11 := fmt.Sprintf("lnbc%dm1pwxxxxxxx", int64(amount/1000))

	result = map[string]any{
		"type":         "incoming",
		"invoice":      bolt11,
		"description":  description,
		"payment_hash": hex.Enc(paymentHash),
		"amount":       int64(amount),
		"created_at":   time.Now().Unix(),
		"expires_at":   time.Now().Add(24 * time.Hour).Unix(),
	}
	return
}

// payInvoice pays a Lightning invoice
func (m *MockWalletService) payInvoice(params any) (result map[string]any, err error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		err = fmt.Errorf("invalid params")
		return
	}

	invoice, ok := paramsMap["invoice"].(string)
	if !ok {
		err = fmt.Errorf("missing or invalid invoice")
		return
	}

	// Mock payment amount (would parse from invoice in real implementation)
	amount := int64(1000) // 1000 msats

	// Check balance
	m.balanceMutex.Lock()
	if m.balance*1000 < amount {
		m.balanceMutex.Unlock()
		err = fmt.Errorf("insufficient balance")
		return
	}
	m.balance -= amount / 1000
	m.balanceMutex.Unlock()

	preimage := make([]byte, 32)
	rand.Read(preimage)

	result = map[string]any{
		"type":       "outgoing",
		"invoice":    invoice,
		"amount":     amount,
		"preimage":   hex.Enc(preimage),
		"created_at": time.Now().Unix(),
	}

	// Emit payment_sent notification
	go m.emitPaymentNotification("payment_sent", result)
	return
}

// sendSuccessResponse sends a successful NWC response
func (m *MockWalletService) sendSuccessResponse(clientPubkey []byte, conversationKey []byte, result any) (err error) {
	response := map[string]any{
		"result": result,
	}

	var responseBytes []byte
	if responseBytes, err = json.Marshal(response); chk.E(err) {
		return
	}

	return m.sendEncryptedResponse(clientPubkey, conversationKey, responseBytes)
}

// sendErrorResponse sends an error NWC response
func (m *MockWalletService) sendErrorResponse(clientPubkey []byte, conversationKey []byte, code, message string) (err error) {
	response := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}

	var responseBytes []byte
	if responseBytes, err = json.Marshal(response); chk.E(err) {
		return
	}

	return m.sendEncryptedResponse(clientPubkey, conversationKey, responseBytes)
}

// sendEncryptedResponse sends an encrypted response event (kind 23195)
func (m *MockWalletService) sendEncryptedResponse(clientPubkey []byte, conversationKey []byte, content []byte) (err error) {
	var encrypted []byte
	if encrypted, err = encryption.Encrypt(content, conversationKey); chk.E(err) {
		return
	}

	ev := &event.E{
		Content:   encrypted,
		CreatedAt: timestamp.Now(),
		Kind:      kind.New(23195),
		Tags: tags.New(
			tag.New("encryption", "nip44_v2"),
			tag.New("p", hex.Enc(clientPubkey)),
		),
	}

	if err = ev.Sign(m.walletSecretKey); chk.E(err) {
		return
	}

	return m.client.Publish(m.ctx, ev)
}

// emitPaymentNotification emits a payment notification (kind 23197)
func (m *MockWalletService) emitPaymentNotification(notificationType string, paymentData map[string]any) (err error) {
	notification := map[string]any{
		"notification_type": notificationType,
		"notification":      paymentData,
	}

	var content []byte
	if content, err = json.Marshal(notification); chk.E(err) {
		return
	}

	// Send notification to all connected clients
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	for clientPubkeyHex, conversationKey := range m.connectedClients {
		var clientPubkey []byte
		if clientPubkey, err = hex.Dec(clientPubkeyHex); chk.E(err) {
			continue
		}

		var encrypted []byte
		if encrypted, err = encryption.Encrypt(content, conversationKey); chk.E(err) {
			continue
		}

		ev := &event.E{
			Content:   encrypted,
			CreatedAt: timestamp.Now(),
			Kind:      kind.New(23197),
			Tags: tags.New(
				tag.New("encryption", "nip44_v2"),
				tag.New("p", hex.Enc(clientPubkey)),
			),
		}

		if err = ev.Sign(m.walletSecretKey); chk.E(err) {
			continue
		}

		m.client.Publish(m.ctx, ev)
	}
	return
}

// SimulateIncomingPayment simulates an incoming payment for testing
func (m *MockWalletService) SimulateIncomingPayment(pubkey []byte, amount int64, description string) (err error) {
	// Add to balance
	m.balanceMutex.Lock()
	m.balance += amount / 1000 // convert msats to sats
	m.balanceMutex.Unlock()

	paymentHash := make([]byte, 32)
	rand.Read(paymentHash)

	preimage := make([]byte, 32)
	rand.Read(preimage)

	paymentData := map[string]any{
		"type":         "incoming",
		"invoice":      fmt.Sprintf("lnbc%dm1pwxxxxxxx", amount/1000),
		"description":  description,
		"amount":       amount,
		"payment_hash": hex.Enc(paymentHash),
		"preimage":     hex.Enc(preimage),
		"created_at":   time.Now().Unix(),
	}

	// Emit payment_received notification
	return m.emitPaymentNotification("payment_received", paymentData)
}
