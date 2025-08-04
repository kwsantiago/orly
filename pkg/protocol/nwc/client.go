package nwc

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	"orly.dev/pkg/utils/log"
	"strings"
	"sync"
	"time"
)

// Options represents options for a NWC client
type Options struct {
	RelayURL     string
	Secret       signer.I
	WalletPubkey []byte
	Lud16        string
}

// Client represents a NWC client
type Client struct {
	options Options
	relay   *ws.Client
	mu      sync.Mutex
}

// ParseWalletConnectURL parses a wallet connect URL
func ParseWalletConnectURL(walletConnectURL string) (opts *Options, err error) {
	if !strings.HasPrefix(walletConnectURL, "nostr+walletconnect://") {
		return nil, fmt.Errorf("unexpected scheme. Should be nostr+walletconnect://")
	}
	// Parse URL
	colonIndex := strings.Index(walletConnectURL, ":")
	if colonIndex == -1 {
		err = fmt.Errorf("invalid URL format")
		return
	}
	walletConnectURL = walletConnectURL[colonIndex+1:]
	if strings.HasPrefix(walletConnectURL, "//") {
		walletConnectURL = walletConnectURL[2:]
	}
	walletConnectURL = "https://" + walletConnectURL
	var u *url.URL
	if u, err = url.Parse(walletConnectURL); chk.E(err) {
		err = fmt.Errorf("failed to parse URL: %w", err)
		return
	}
	// Get wallet pubkey
	walletPubkey := u.Host
	if len(walletPubkey) != 64 {
		err = fmt.Errorf("incorrect wallet pubkey found in auth string")
		return
	}
	var pk []byte
	if pk, err = hex.Dec(walletPubkey); chk.E(err) {
		err = fmt.Errorf("failed to decode pubkey: %w", err)
		return
	}
	// Get relay URL
	relayURL := u.Query().Get("relay")
	if relayURL == "" {
		return nil, fmt.Errorf("no relay URL found in auth string")
	}
	// Get secret
	secret := u.Query().Get("secret")
	if secret == "" {
		return nil, fmt.Errorf("no secret found in auth string")
	}
	var sk []byte
	if sk, err = hex.Dec(secret); chk.E(err) {
		return
	}
	sign := &p256k.Signer{}
	if err = sign.InitSec(sk); chk.E(err) {
		return
	}
	opts = &Options{
		RelayURL:     relayURL,
		Secret:       sign,
		WalletPubkey: pk,
	}
	return
}

// NewNWCClient creates a new NWC client
func NewNWCClient(options *Options) (cl *Client, err error) {
	if options.RelayURL == "" {
		err = fmt.Errorf("missing relay URL")
		return
	}
	if options.Secret == nil {
		err = fmt.Errorf("missing secret")
		return
	}
	if options.WalletPubkey == nil {
		err = fmt.Errorf("missing wallet pubkey")
		return
	}
	return &Client{
		options: Options{
			RelayURL:     options.RelayURL,
			Secret:       options.Secret,
			WalletPubkey: options.WalletPubkey,
			Lud16:        options.Lud16,
		},
	}, nil
}

// NostrWalletConnectURL returns the nostr wallet connect URL
func (c *Client) NostrWalletConnectURL() string {
	return c.GetNostrWalletConnectURL(true)
}

// GetNostrWalletConnectURL returns the nostr wallet connect URL
func (c *Client) GetNostrWalletConnectURL(includeSecret bool) string {
	params := url.Values{}
	params.Add("relay", c.options.RelayURL)
	if includeSecret {
		params.Add("secret", hex.Enc(c.options.Secret.Sec()))
	}
	return fmt.Sprintf(
		"nostr+walletconnect://%s?%s", c.options.WalletPubkey, params.Encode(),
	)
}

// Connected returns whether the client is connected to the relay
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.relay != nil && c.relay.IsConnected()
}

// GetPublicKey returns the client's public key
func (c *Client) GetPublicKey() (pubkey []byte, err error) {
	pubkey = c.options.Secret.Pub()
	return
}

// Close closes the relay connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.relay != nil {
		c.relay.Close()
		c.relay = nil
	}
}

// Encrypt encrypts content for a pubkey
func (c *Client) encrypt(pubkey, content []byte) (
	cipherText []byte, err error,
) {
	var sharedSecret []byte
	if sharedSecret, err = c.options.Secret.ECDH(pubkey); chk.E(err) {
		return
	}
	cipherText, err = encryption.EncryptNip4(content, sharedSecret)
	return
}

// Decrypt decrypts content from a pubkey
func (c *Client) decrypt(pubkey, content []byte) (plaintext []byte, err error) {
	var sharedSecret []byte
	if sharedSecret, err = c.options.Secret.ECDH(pubkey); chk.E(err) {
		return
	}
	plaintext, err = encryption.DecryptNip4(content, sharedSecret)
	return
}

// GetInfo gets wallet info
func (c *Client) GetInfo() (response *GetInfoResponse, err error) {
	var result []byte
	if result, err = c.executeRequest(GetInfo, nil); chk.E(err) {
		return
	}
	response = &GetInfoResponse{}
	if err = json.Unmarshal(result, response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// GetBudget gets wallet budget
func (c *Client) GetBudget() (response *GetBudgetResponse, err error) {
	var result []byte
	result, err = c.executeRequest(GetBudget, nil)
	if err != nil {
		return nil, err
	}
	response = &GetBudgetResponse{}
	if err = json.Unmarshal(result, response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// GetBalance gets wallet balance
func (c *Client) GetBalance() (response *GetBalanceResponse, err error) {
	var result []byte
	if result, err = c.executeRequest(GetBalance, nil); chk.E(err) {
		return
	}
	response = &GetBalanceResponse{}
	if err = json.Unmarshal(result, response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// PayInvoice pays an invoice
func (c *Client) PayInvoice(request *PayInvoiceRequest) (
	response *PayResponse, err error,
) {
	var result []byte
	result, err = c.executeRequest(PayInvoice, request)
	if err != nil {
		return nil, err
	}
	response = &PayResponse{}
	if err = json.Unmarshal(result, response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// PayKeysend sends a keysend payment
func (c *Client) PayKeysend(request *PayKeysendRequest) (
	response *PayResponse, err error,
) {
	var result []byte
	if result, err = c.executeRequest(PayKeysend, request); chk.E(err) {
		return
	}
	response = &PayResponse{}
	if err = json.Unmarshal(result, response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// MakeInvoice creates an invoice
func (c *Client) MakeInvoice(request *MakeInvoiceRequest) (
	response *Transaction, err error,
) {
	var result []byte
	if result, err = c.executeRequest(MakeInvoice, request); chk.E(err) {
		return
	}
	response = &Transaction{}
	if err = json.Unmarshal(result, response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return
}

// LookupInvoice looks up an invoice
func (c *Client) LookupInvoice(request *LookupInvoiceRequest) (
	response *Transaction, err error,
) {
	var result []byte
	if result, err = c.executeRequest(LookupInvoice, request); chk.E(err) {
		return
	}
	response = &Transaction{}
	if err = json.Unmarshal(result, response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// ListTransactions lists transactions
func (c *Client) ListTransactions(request *ListTransactionsRequest) (
	response *ListTransactionsResponse, err error,
) {
	var result []byte
	if result, err = c.executeRequest(ListTransactions, request); chk.E(err) {
		return
	}
	response = &ListTransactionsResponse{}
	if err = json.Unmarshal(result, response); chk.E(err) {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// SignMessage signs a message
func (c *Client) SignMessage(request *SignMessageRequest) (
	response *SignMessageResponse, err error,
) {
	var result []byte
	if result, err = c.executeRequest(SignMessage, request); chk.E(err) {
		return
	}
	response = &SignMessageResponse{}
	if err = json.Unmarshal(result, response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		return
	}
	return
}

// NotificationHandler is a function that handles notifications
type NotificationHandler func(*Notification)

// SubscribeNotifications subscribes to notifications
func (c *Client) SubscribeNotifications(
	handler NotificationHandler,
	notificationTypes []NotificationType,
) (stop func(), err error) {
	if handler == nil {
		err = fmt.Errorf("missing notification handler")
		return
	}
	ctx, cancel := context.Cancel(context.Bg())
	doneCh := make(chan struct{})
	stop = func() {
		cancel()
		<-doneCh
	}
	go func() {
		defer close(doneCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Check connection
				if err := c.checkConnected(); err != nil {
					time.Sleep(1 * time.Second)
					continue
				}
				// Get client pubkey
				var clientPubkey []byte
				if clientPubkey, err = c.GetPublicKey(); chk.E(err) {
					time.Sleep(1 * time.Second)
					continue
				}
				// Subscribe to events
				f := &filter.F{
					Kinds:   kinds.New(kind.WalletResponse),
					Authors: tag.New(c.options.WalletPubkey),
					Tags:    tags.New(tag.New([]byte("#p"), clientPubkey)),
				}
				var sub *ws.Subscription
				if sub, err = c.relay.Subscribe(
					context.Bg(), &filters.T{
						F: []*filter.F{f},
					},
				); chk.E(err) {
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
						// Decrypt content
						var decryptedContent []byte
						if decryptedContent, err = c.decrypt(
							c.options.WalletPubkey, ev.Content,
						); chk.E(err) {
							log.E.F(
								"Failed to decrypt event content: %v\n", err,
							)
							continue
						}
						// Parse notification
						notification := &Notification{}
						if err = json.Unmarshal(
							decryptedContent, notification,
						); chk.E(err) {
							log.E.F(
								"Failed to parse notification: %v\n", err,
							)
							continue
						}
						// Check if notification type is requested
						if len(notificationTypes) > 0 {
							found := false
							for _, t := range notificationTypes {
								if notification.NotificationType == t {
									found = true
									break
								}
							}
							if !found {
								continue
							}
						}
						// Handle notification
						handler(notification)
					case <-sub.EndOfStoredEvents:
						// Ignore
					}
				}
			}
		}
	}()
	return
}

// executeRequest executes a NIP-47 request
func (c *Client) executeRequest(
	method Method,
	params any,
) (msg json.RawMessage, err error) {
	// Default timeout values
	replyTimeout := 3 * time.Second
	publishTimeout := 3 * time.Second
	// Create context with timeout
	ctx, cancel := context.Timeout(context.Bg(), replyTimeout)
	defer cancel()
	// Create result channel
	resultCh := make(chan json.RawMessage, 1)
	errCh := make(chan error, 1)
	// Check connection
	if err = c.checkConnected(); err != nil {
		return nil, err
	}
	// Create request
	request := struct {
		Method Method `json:"method"`
		Params any    `json:"params,omitempty"`
	}{
		Method: method,
		Params: params,
	}
	// Marshal request
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	// Encrypt request
	var encryptedContent []byte
	if encryptedContent, err = c.encrypt(
		c.options.WalletPubkey, requestJSON,
	); chk.E(err) {
		return nil, fmt.Errorf("failed to encrypt request: %w", err)
	}
	// Create request event
	requestEvent := &event.E{
		Kind:      kind.WalletRequest,
		CreatedAt: timestamp.New(time.Now().Unix()),
		Tags:      tags.New(tag.New("p", hex.Enc(c.options.WalletPubkey))),
		Content:   encryptedContent,
	}
	// Sign request event
	err = requestEvent.Sign(c.options.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request event: %w", err)
	}
	// Subscribe to response events
	f := &filter.F{
		Kinds:   kinds.New(kind.WalletResponse),
		Authors: tag.New(c.options.WalletPubkey),
		Tags:    tags.New(tag.New([]byte("#p"), requestEvent.ID)),
	}
	log.I.F("%s", f.Marshal(nil))
	var sub *ws.Subscription
	if sub, err = c.relay.Subscribe(
		ctx, &filters.T{
			F: []*filter.F{f},
		},
	); chk.E(err) {
		err = fmt.Errorf(
			"failed to subscribe to response events: %w", err,
		)
		return
	}
	defer sub.Close()
	// Set up reply timeout
	replyTimer := time.AfterFunc(
		replyTimeout, func() {
			errCh <- NewReplyTimeoutError(
				fmt.Sprintf("Timeout waiting for reply to %s", method),
				"TIMEOUT",
			)
		},
	)
	defer replyTimer.Stop()
	// Handle response events
	go func() {
		var resErr error
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-sub.Events:
				// Decrypt content
				var decryptedContent []byte
				decryptedContent, resErr = c.decrypt(
					c.options.WalletPubkey, ev.Content,
				)
				if chk.E(resErr) {
					errCh <- fmt.Errorf(
						"failed to decrypt response: %w",
						resErr,
					)
					return
				}
				// Parse response
				var response struct {
					ResultType string          `json:"result_type"`
					Result     json.RawMessage `json:"result"`
					Error      *struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"error"`
				}
				if resErr = json.Unmarshal(
					decryptedContent, &response,
				); chk.E(resErr) {
					errCh <- fmt.Errorf("failed to parse response: %w", resErr)
					return
				}
				// Check for error
				if response.Error != nil {
					errCh <- NewWalletError(
						response.Error.Message,
						response.Error.Code,
					)
					return
				}
				// Send result
				resultCh <- response.Result
				return
			case <-sub.EndOfStoredEvents:
				// Ignore
			}
		}
	}()
	// Publish request event
	publishCtx, publishCancel := context.Timeout(
		context.Bg(), publishTimeout,
	)
	defer publishCancel()
	if err = c.relay.Publish(publishCtx, requestEvent); chk.E(err) {
		err = fmt.Errorf("failed to publish request event: %w", err)
		return
	}

	// Wait for result or error
	select {
	case msg = <-resultCh:
		return
	case err = <-errCh:
		return
	case <-ctx.Done():
		err = NewReplyTimeoutError(
			fmt.Sprintf("Timeout waiting for reply to %s", method),
			"TIMEOUT",
		)
		return
	}
}

// checkConnected checks if the client is connected to the relay
func (c *Client) checkConnected() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.options.RelayURL == "" {
		return fmt.Errorf("missing relay URL")
	}

	if c.relay == nil {
		if c.relay, err = ws.RelayConnect(
			context.Bg(), c.options.RelayURL,
		); chk.E(err) {
			return NewNetworkError(
				"Failed to connect to "+c.options.RelayURL,
				"OTHER",
			)
		}
	} else if !c.relay.IsConnected() {
		c.relay.Close()
		if c.relay, err = ws.RelayConnect(
			context.Bg(), c.options.RelayURL,
		); chk.E(err) {
			return NewNetworkError(
				"Failed to connect to "+c.options.RelayURL,
				"OTHER",
			)
		}
	}
	return nil
}
