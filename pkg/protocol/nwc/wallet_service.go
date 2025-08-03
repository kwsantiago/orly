package nwc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"orly.dev/pkg/crypto/encryption"
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
	"orly.dev/pkg/utils/log"
	"sync"
	"time"
)

// WalletServiceKeyPair represents a key pair for a wallet service
type WalletServiceKeyPair struct {
	WalletKey    signer.I
	ClientPubkey []byte
}

// NewWalletServiceKeyPair creates a new WalletServiceKeyPair
func NewWalletServiceKeyPair(walletKey signer.I, clientPubkey []byte) (
	*WalletServiceKeyPair, error,
) {
	if walletKey == nil {
		return nil, fmt.Errorf("missing wallet secret key")
	}
	if len(clientPubkey) == 0 {
		return nil, fmt.Errorf("missing client pubkey")
	}
	return &WalletServiceKeyPair{walletKey, clientPubkey}, nil
}

// NewWalletServiceOptions represents options for creating a new wallet service
type NewWalletServiceOptions struct {
	RelayURL string
}

// WalletService represents a wallet service
type WalletService struct {
	relay    *ws.Client
	relayURL string
	mu       sync.Mutex
}

// NewWalletService creates a new WalletService
func NewWalletService(options *NewWalletServiceOptions) (
	*WalletService, error,
) {
	if options.RelayURL == "" {
		return nil, fmt.Errorf("missing relay URL")
	}
	return &WalletService{
		relayURL: options.RelayURL,
	}, nil
}

// PublishWalletServiceInfoEvent publishes a wallet service info event
func (s *WalletService) PublishWalletServiceInfoEvent(
	walletSecret signer.I,
	supportedMethods []Method,
	supportedNotifications []NotificationType,
) (err error) {
	if err = s.checkConnected(); err != nil {
		return
	}
	// Convert methods to space-separated string
	var methodsStr []byte
	for i, method := range supportedMethods {
		if i > 0 {
			methodsStr = append(methodsStr, ' ')
		}
		methodsStr = append(methodsStr, method...)
	}
	// Convert notifications to tags
	notificationsTag := tag.New("notifications")
	for _, notification := range supportedNotifications {
		notificationsTag.Append(notification)
	}
	// Create event
	ev := &event.E{
		Kind:      kind.New(13194),
		CreatedAt: timestamp.New(time.Now().Unix()),
		Tags: tags.New(
			tag.New("encryption", "nip04 nip44_v2"),
			notificationsTag,
		),
		Content: methodsStr,
	}
	// Sign event
	if err = ev.Sign(walletSecret); chk.E(err) {
		return fmt.Errorf("failed to sign ev: %w", err)
	}
	// Publish event
	if err = s.relay.Publish(context.Background(), ev); chk.E(err) {
		return fmt.Errorf(
			"failed to publish wallet service info ev: %w", err,
		)
	}
	return
}

// Subscribe subscribes to client requests
func (s *WalletService) Subscribe(
	keypair *WalletServiceKeyPair,
	handler WalletServiceRequestHandler,
) (func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Check connection
				if err := s.checkConnected(); err != nil {
					errCh <- err
					time.Sleep(1 * time.Second)
					continue
				}
				f := &filter.F{
					Kinds:   kinds.New(kind.New(23194)),
					Authors: tag.New(keypair.ClientPubkey),
					Tags: tags.New(
						tag.New(
							"p", hex.Enc(keypair.WalletKey.Pub()),
						),
					),
				}
				// Subscribe to events
				sub, err := s.relay.Subscribe(
					context.Background(), &filters.T{
						F: []*filter.F{f},
					},
				)
				if err != nil {
					errCh <- fmt.Errorf("failed to subscribe: %w", err)
					time.Sleep(1 * time.Second)
					continue
				}
				// Handle events
				for {
					select {
					case <-ctx.Done():
						sub.Close()
						return
					case ev := <-sub.Events:
						go s.handleEvent(ev, keypair, handler)
					case <-sub.EndOfStoredEvents:
						// Ignore
					}
				}
			}
		}
	}()
	return func() {
		cancel()
		<-doneCh
	}, nil
}

// handleEvent handles a client request event
func (s *WalletService) handleEvent(
	ev *event.E,
	keypair *WalletServiceKeyPair,
	handler WalletServiceRequestHandler,
) {
	// Get encryption type
	encryptionType := Nip04
	for _, tag := range ev.Tags.ToSliceOfTags() {
		if tag.Len() >= 2 && bytes.Equal(tag.B(0), []byte("encryption")) {
			if bytes.Equal(tag.Value(), []byte("nip44_v2")) {
				encryptionType = Nip44V2
			}
			break
		}
	}
	var err error
	// Decrypt content
	var decryptedContent []byte
	if decryptedContent, err = s.decrypt(
		keypair, ev.Content, encryptionType,
	); chk.E(err) {
		return
	}
	// Parse request
	var request struct {
		Method Method          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err = json.Unmarshal([]byte(decryptedContent), &request); chk.E(err) {
		return
	}
	// Handle request
	var response *WalletServiceResponse
	switch request.Method {
	case GetInfo:
		response, err = handler.GetInfo()
	case MakeInvoice:
		var params MakeInvoiceRequest
		if err = json.Unmarshal(request.Params, &params); chk.E(err) {
			return
		}
		response, err = handler.MakeInvoice(&params)
	case PayInvoice:
		var params PayInvoiceRequest
		if err = json.Unmarshal(request.Params, &params); chk.E(err) {
			return
		}
		response, err = handler.PayInvoice(&params)
	case PayKeysend:
		var params PayKeysendRequest
		if err = json.Unmarshal(request.Params, &params); chk.E(err) {
			return
		}
		response, err = handler.PayKeysend(&params)
	case GetBalance:
		response, err = handler.GetBalance()
	case LookupInvoice:
		var params LookupInvoiceRequest
		if err = json.Unmarshal(request.Params, &params); chk.E(err) {
			log.E.F("Failed to parse lookup_invoice params: %v\n", err)
			return
		}
		response, err = handler.LookupInvoice(&params)
	case ListTransactions:
		var params ListTransactionsRequest
		if err = json.Unmarshal(request.Params, &params); chk.E(err) {
			return
		}
		response, err = handler.ListTransactions(&params)
	case SignMessage:
		var params SignMessageRequest
		if err = json.Unmarshal(request.Params, &params); chk.E(err) {
			return
		}
		response, err = handler.SignMessage(&params)
	default:
		// Unsupported method
		response = &WalletServiceResponse{
			Error: &WalletServiceRequestHandlerError{
				Code:    "NOT_IMPLEMENTED",
				Message: "This method is not supported by the wallet service",
			},
		}
	}
	if chk.E(err) {
		return
	}
	if response == nil {
		log.E.F("Received unsupported method: %s\n", request.Method)
		response = &WalletServiceResponse{
			Error: &WalletServiceRequestHandlerError{
				Code:    "NOT_IMPLEMENTED",
				Message: "This method is not supported by the wallet service",
			},
		}
	}
	// Create response
	responseData := struct {
		ResultType string                            `json:"result_type"`
		Result     interface{}                       `json:"result,omitempty"`
		Error      *WalletServiceRequestHandlerError `json:"error,omitempty"`
	}{
		ResultType: string(request.Method),
		Result:     response.Result,
		Error:      response.Error,
	}
	// Encrypt response
	var responseJSON []byte
	if responseJSON, err = json.Marshal(responseData); chk.E(err) {
		log.E.F("Failed to marshal response: %v\n", err)
		return
	}
	var encryptedContent []byte
	if encryptedContent, err = s.encrypt(
		keypair, responseJSON, encryptionType,
	); chk.E(err) {
		log.E.F("Failed to encrypt response: %v\n", err)
		return
	}
	// Create response event
	responseEvent := &event.E{
		Kind:      kind.New(23195),
		CreatedAt: timestamp.New(time.Now().Unix()),
		Tags:      tags.New(tag.New([]byte("e"), ev.ID)),
		Content:   encryptedContent,
	}
	// Sign response event
	if err = responseEvent.Sign(keypair.WalletKey); chk.E(err) {
		return
	}
	// Publish response event
	err = s.relay.Publish(context.Background(), responseEvent)
	if err != nil {
		return
	}
}

// Connected returns whether the relay is connected
func (s *WalletService) Connected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.relay != nil && s.relay.IsConnected()
}

// Close closes the relay connection
func (s *WalletService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.relay != nil {
		s.relay.Close()
		s.relay = nil
	}
}

// encrypt encrypts content using the specified encryption type
func (s *WalletService) encrypt(
	keypair *WalletServiceKeyPair,
	content []byte,
	encryptionType EncryptionType,
) (message []byte, err error) {
	var key []byte
	if key, err = keypair.WalletKey.ECDH(keypair.ClientPubkey); chk.E(err) {
		return
	}
	switch encryptionType {
	case Nip04:
		return encryption.DecryptNip4(content, key)
	case Nip44V2:
		message, err = encryption.Encrypt(content, key)
		if err != nil {
			err = fmt.Errorf("failed to encrypt with nip44: %w", err)
			return
		}
		return
	default:
		err = fmt.Errorf("unsupported encryption type: %s", encryptionType)
		return
	}
}

// decrypt decrypts content using the specified encryption type
func (s *WalletService) decrypt(
	keypair *WalletServiceKeyPair,
	content []byte,
	encryptionType EncryptionType,
) (message []byte, err error) {
	var key []byte
	if key, err = keypair.WalletKey.ECDH(keypair.ClientPubkey); chk.E(err) {
		return
	}
	switch encryptionType {
	case Nip04:
		return encryption.DecryptNip4(key, content)
	case Nip44V2:
		message, err = encryption.Decrypt(content, key)
		if err != nil {
			err = fmt.Errorf("failed to decrypt with nip44: %w", err)
			return
		}
		return
	default:
		err = fmt.Errorf("unsupported encryption type: %s", encryptionType)
		return
	}
}

// checkConnected checks if the relay is connected and connects if not
func (s *WalletService) checkConnected() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.relayURL == "" {
		return fmt.Errorf("missing relay URL")
	}

	if s.relay == nil {
		if s.relay, err = ws.RelayConnect(
			context.Background(), s.relayURL,
		); chk.E(err) {
			return NewNetworkError(
				"Failed to connect to "+s.relayURL,
				"OTHER",
			)
		}
	} else if !s.relay.IsConnected() {
		s.relay.Close()
		if s.relay, err = ws.RelayConnect(
			context.Background(), s.relayURL,
		); chk.E(err) {
			return NewNetworkError(
				"Failed to connect to "+s.relayURL,
				"OTHER",
			)
		}
	}
	return nil
}
