package nwc

import (
	"encoding/json"
	"fmt"
	"sync"

	"orly.dev/pkg/crypto/encryption"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
)

// WalletService represents a wallet service that clients can connect to.
type WalletService struct {
	mutex           sync.Mutex
	listener        *ws.Listener
	walletSecretKey signer.I
	walletPublicKey []byte
	conversationKey []byte // nip44
	handlers        map[string]MethodHandler
}

// MethodHandler is a function type for handling wallet service method calls.
type MethodHandler func(
	c context.T, params json.RawMessage,
) (result interface{}, err error)

// NewWalletService creates a new WalletService with the given listener and wallet key.
func NewWalletService(
	listener *ws.Listener, walletKey signer.I,
) (ws *WalletService, err error) {
	pubKey := walletKey.Pub()

	ws = &WalletService{
		listener:        listener,
		walletSecretKey: walletKey,
		walletPublicKey: pubKey,
		handlers:        make(map[string]MethodHandler),
	}

	// Register default method handlers
	ws.registerDefaultHandlers()

	return
}

// RegisterHandler registers a handler for a specific method.
func (ws *WalletService) RegisterHandler(
	method string, handler MethodHandler,
) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.handlers[method] = handler
}

// registerDefaultHandlers registers the default empty stub handlers for all supported methods.
func (ws *WalletService) registerDefaultHandlers() {
	// Register handlers for all supported methods
	ws.RegisterHandler(string(GetWalletServiceInfo), ws.handleGetWalletServiceInfo)
	ws.RegisterHandler(string(CancelHoldInvoice), ws.handleCancelHoldInvoice)
	ws.RegisterHandler(string(CreateConnection), ws.handleCreateConnection)
	ws.RegisterHandler(string(GetBalance), ws.handleGetBalance)
	ws.RegisterHandler(string(GetBudget), ws.handleGetBudget)
	ws.RegisterHandler(string(GetInfo), ws.handleGetInfo)
	ws.RegisterHandler(string(ListTransactions), ws.handleListTransactions)
	ws.RegisterHandler(string(LookupInvoice), ws.handleLookupInvoice)
	ws.RegisterHandler(string(MakeHoldInvoice), ws.handleMakeHoldInvoice)
	ws.RegisterHandler(string(MakeInvoice), ws.handleMakeInvoice)
	ws.RegisterHandler(string(PayKeysend), ws.handlePayKeysend)
	ws.RegisterHandler(string(PayInvoice), ws.handlePayInvoice)
	ws.RegisterHandler(string(SettleHoldInvoice), ws.handleSettleHoldInvoice)
	ws.RegisterHandler(string(SignMessage), ws.handleSignMessage)
}

// HandleRequest processes an incoming wallet request event.
func (ws *WalletService) HandleRequest(c context.T, ev *event.E) (err error) {
	// Verify the event is a wallet request
	if ev.Kind != kind.WalletRequest {
		return fmt.Errorf("invalid event kind: %d", ev.Kind)
	}

	// Get the client's public key from the event
	clientPubKey := ev.Pubkey

	// Generate conversation key
	var ck []byte
	if ck, err = encryption.GenerateConversationKeyWithSigner(
		ws.walletSecretKey,
		clientPubKey,
	); chk.E(err) {
		return
	}

	// Decrypt the content
	var content []byte
	if content, err = encryption.Decrypt(ev.Content, ck); chk.E(err) {
		return
	}

	// Parse the request
	var req Request
	if err = json.Unmarshal(content, &req); chk.E(err) {
		return
	}

	// Find the handler for the method
	ws.mutex.Lock()
	handler, exists := ws.handlers[req.Method]
	ws.mutex.Unlock()

	var result interface{}
	var respErr *ResponseError

	if !exists {
		respErr = &ResponseError{
			Code:    "method_not_found",
			Message: fmt.Sprintf("method %s not found", req.Method),
		}
	} else {
		// Call the handler
		var params json.RawMessage
		if req.Params != nil {
			var paramsBytes []byte
			if paramsBytes, err = json.Marshal(req.Params); chk.E(err) {
				return
			}
			params = paramsBytes
		}

		result, err = handler(c, params)
		if err != nil {
			if re, ok := err.(*ResponseError); ok {
				respErr = re
			} else {
				respErr = &ResponseError{
					Code:    "internal_error",
					Message: err.Error(),
				}
			}
		}
	}

	// Create response
	resp := Response{
		ResultType: req.Method,
		Result:     result,
		Error:      respErr,
	}

	// Marshal response
	var respBytes []byte
	if respBytes, err = json.Marshal(resp); chk.E(err) {
		return
	}

	// Encrypt response
	var encResp []byte
	if encResp, err = encryption.Encrypt(respBytes, ck); chk.E(err) {
		return
	}

	// Create response event
	respEv := &event.E{
		Content:   encResp,
		CreatedAt: timestamp.Now(),
		Kind:      kind.WalletResponse,
		Tags: tags.New(
			tag.New("p", hex.Enc(clientPubKey)),
			tag.New("e", hex.Enc(ev.ID)),
			tag.New(EncryptionTag, Nip44V2),
		),
	}

	// Sign the response event
	if err = respEv.Sign(ws.walletSecretKey); chk.E(err) {
		return
	}

	// Send the response
	_, err = ws.listener.Write(respEv.Marshal(nil))
	return
}

// SendNotification sends a notification to a client.
func (ws *WalletService) SendNotification(
	c context.T, clientPubKey []byte, notificationType string,
	content interface{},
) (err error) {
	// Generate conversation key
	var ck []byte
	if ck, err = encryption.GenerateConversationKeyWithSigner(
		ws.walletSecretKey,
		clientPubKey,
	); chk.E(err) {
		return
	}

	// Marshal content
	var contentBytes []byte
	if contentBytes, err = json.Marshal(content); chk.E(err) {
		return
	}

	// Encrypt content
	var encContent []byte
	if encContent, err = encryption.Encrypt(contentBytes, ck); chk.E(err) {
		return
	}

	// Create notification event
	notifEv := &event.E{
		Content:   encContent,
		CreatedAt: timestamp.Now(),
		Kind:      kind.WalletNotification,
		Tags: tags.New(
			tag.New("p", hex.Enc(clientPubKey)),
			tag.New(NotificationTag, []byte(notificationType)),
			tag.New(EncryptionTag, Nip44V2),
		),
	}

	// Sign the notification event
	if err = notifEv.Sign(ws.walletSecretKey); chk.E(err) {
		return
	}

	// Send the notification
	_, err = ws.listener.Write(notifEv.Marshal(nil))
	return
}
