package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
	"orly.dev/pkg/protocol/nwc"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/interrupt"
)

var (
	relayURL    = flag.String("relay", "ws://localhost:8080", "Relay URL to connect to")
	walletKey   = flag.String("key", "", "Wallet private key (hex)")
	generateKey = flag.Bool("generate-key", false, "Generate a new wallet key")
)

func main() {
	flag.Parse()

	// Create context
	c, cancel := context.Cancel(context.Bg())
	interrupt.AddHandler(cancel)
	defer cancel()

	// Initialize wallet key
	var walletSigner signer.I
	var err error

	if *generateKey {
		// Generate a new wallet key
		walletSigner = &p256k.Signer{}
		if err = walletSigner.Generate(); chk.E(err) {
			fmt.Printf("Error generating wallet key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated wallet key: %s\n", hex.Enc(walletSigner.Sec()))
		fmt.Printf("Wallet public key: %s\n", hex.Enc(walletSigner.Pub()))
	} else if *walletKey != "" {
		// Use provided wallet key
		if walletSigner, err = p256k.NewSecFromHex(*walletKey); chk.E(err) {
			fmt.Printf("Error initializing wallet key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Using wallet key: %s\n", *walletKey)
		fmt.Printf("Wallet public key: %s\n", hex.Enc(walletSigner.Pub()))
	} else {
		// Generate a temporary wallet key
		walletSigner = &p256k.Signer{}
		if err = walletSigner.Generate(); chk.E(err) {
			fmt.Printf("Error generating temporary wallet key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated temporary wallet key: %s\n", hex.Enc(walletSigner.Sec()))
		fmt.Printf("Wallet public key: %s\n", hex.Enc(walletSigner.Pub()))
	}

	// Connect to relay
	fmt.Printf("Connecting to relay: %s\n", *relayURL)
	relay, err := ws.RelayConnect(c, *relayURL)
	if err != nil {
		fmt.Printf("Error connecting to relay: %v\n", err)
		os.Exit(1)
	}
	defer relay.Close()
	fmt.Println("Connected to relay")

	// Create a mock wallet service info event
	walletServiceInfoEvent := createWalletServiceInfoEvent(walletSigner)

	// Publish wallet service info event
	if err = relay.Publish(c, walletServiceInfoEvent); chk.E(err) {
		fmt.Printf("Error publishing wallet service info: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Published wallet service info")

	// Subscribe to wallet requests
	fmt.Println("Subscribing to wallet requests...")
	sub, err := relay.Subscribe(
		c, filters.New(
			&filter.F{
				Kinds: kinds.New(kind.WalletRequest),
				Tags:  tags.New(tag.New("#p", hex.Enc(walletSigner.Pub()))),
			},
		),
	)
	if err != nil {
		fmt.Printf("Error subscribing to wallet requests: %v\n", err)
		os.Exit(1)
	}
	defer sub.Unsub()
	fmt.Println("Subscribed to wallet requests")

	// Process wallet requests
	fmt.Println("Waiting for wallet requests...")
	for {
		select {
		case <-c.Done():
			fmt.Println("Context canceled, exiting")
			return
		case ev := <-sub.Events:
			fmt.Printf("Received wallet request: %s\n", hex.Enc(ev.ID))
			go handleWalletRequest(c, relay, walletSigner, ev)
		}
	}
}

// handleWalletRequest processes a wallet request and sends a response
func handleWalletRequest(c context.T, relay *ws.Client, walletKey signer.I, ev *event.E) {
	// Get the client's public key from the event
	clientPubKey := ev.Pubkey

	// Generate conversation key
	var ck []byte
	var err error
	if ck, err = encryption.GenerateConversationKeyWithSigner(
		walletKey,
		clientPubKey,
	); chk.E(err) {
		fmt.Printf("Error generating conversation key: %v\n", err)
		return
	}

	// Decrypt the content
	var content []byte
	if content, err = encryption.Decrypt(ev.Content, ck); chk.E(err) {
		fmt.Printf("Error decrypting content: %v\n", err)
		return
	}

	// Parse the request
	var req nwc.Request
	if err = json.Unmarshal(content, &req); chk.E(err) {
		fmt.Printf("Error parsing request: %v\n", err)
		return
	}

	fmt.Printf("Handling method: %s\n", req.Method)

	// Process the request based on the method
	var result interface{}
	var respErr *nwc.ResponseError

	switch req.Method {
	case string(nwc.GetWalletServiceInfo):
		result = handleGetWalletServiceInfo()
	case string(nwc.GetInfo):
		result = handleGetInfo(walletKey)
	case string(nwc.GetBalance):
		result = handleGetBalance()
	case string(nwc.GetBudget):
		result = handleGetBudget()
	case string(nwc.MakeInvoice):
		result = handleMakeInvoice()
	case string(nwc.PayInvoice):
		result = handlePayInvoice()
	case string(nwc.PayKeysend):
		result = handlePayKeysend()
	case string(nwc.LookupInvoice):
		result = handleLookupInvoice()
	case string(nwc.ListTransactions):
		result = handleListTransactions()
	case string(nwc.MakeHoldInvoice):
		result = handleMakeHoldInvoice()
	case string(nwc.SettleHoldInvoice):
		// No result for SettleHoldInvoice
	case string(nwc.CancelHoldInvoice):
		// No result for CancelHoldInvoice
	case string(nwc.SignMessage):
		result = handleSignMessage()
	case string(nwc.CreateConnection):
		// No result for CreateConnection
	default:
		respErr = &nwc.ResponseError{
			Code:    "method_not_found",
			Message: fmt.Sprintf("method %s not found", req.Method),
		}
	}

	// Create response
	resp := nwc.Response{
		ResultType: req.Method,
		Result:     result,
		Error:      respErr,
	}

	// Marshal response
	var respBytes []byte
	if respBytes, err = json.Marshal(resp); chk.E(err) {
		fmt.Printf("Error marshaling response: %v\n", err)
		return
	}

	// Encrypt response
	var encResp []byte
	if encResp, err = encryption.Encrypt(respBytes, ck); chk.E(err) {
		fmt.Printf("Error encrypting response: %v\n", err)
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
			tag.New(string(nwc.EncryptionTag), string(nwc.Nip44V2)),
		),
	}

	// Sign the response event
	if err = respEv.Sign(walletKey); chk.E(err) {
		fmt.Printf("Error signing response event: %v\n", err)
		return
	}

	// Publish the response event
	if err = relay.Publish(c, respEv); chk.E(err) {
		fmt.Printf("Error publishing response event: %v\n", err)
		return
	}

	fmt.Printf("Successfully handled request: %s\n", hex.Enc(ev.ID))
}

// createWalletServiceInfoEvent creates a wallet service info event
func createWalletServiceInfoEvent(walletKey signer.I) *event.E {
	ev := &event.E{
		Content: []byte(
			string(nwc.GetWalletServiceInfo) + " " +
				string(nwc.GetInfo) + " " +
				string(nwc.GetBalance) + " " +
				string(nwc.GetBudget) + " " +
				string(nwc.MakeInvoice) + " " +
				string(nwc.PayInvoice) + " " +
				string(nwc.PayKeysend) + " " +
				string(nwc.LookupInvoice) + " " +
				string(nwc.ListTransactions) + " " +
				string(nwc.MakeHoldInvoice) + " " +
				string(nwc.SettleHoldInvoice) + " " +
				string(nwc.CancelHoldInvoice) + " " +
				string(nwc.SignMessage) + " " +
				string(nwc.CreateConnection),
		),
		CreatedAt: timestamp.Now(),
		Kind:      kind.WalletServiceInfo,
		Tags: tags.New(
			tag.New(string(nwc.EncryptionTag), string(nwc.Nip44V2)),
			tag.New(string(nwc.NotificationTag), string(nwc.PaymentReceived)+" "+string(nwc.PaymentSent)+" "+string(nwc.HoldInvoiceAccepted)),
		),
	}
	if err := ev.Sign(walletKey); chk.E(err) {
		fmt.Printf("Error signing wallet service info event: %v\n", err)
		os.Exit(1)
	}
	return ev
}

// Handler functions for each method

func handleGetWalletServiceInfo() *nwc.WalletServiceInfo {
	fmt.Println("Handling GetWalletServiceInfo request")
	return &nwc.WalletServiceInfo{
		EncryptionTypes: []nwc.EncryptionType{nwc.Nip44V2},
		Capabilities: []nwc.Capability{
			nwc.GetWalletServiceInfo,
			nwc.GetInfo,
			nwc.GetBalance,
			nwc.GetBudget,
			nwc.MakeInvoice,
			nwc.PayInvoice,
			nwc.PayKeysend,
			nwc.LookupInvoice,
			nwc.ListTransactions,
			nwc.MakeHoldInvoice,
			nwc.SettleHoldInvoice,
			nwc.CancelHoldInvoice,
			nwc.SignMessage,
			nwc.CreateConnection,
		},
		NotificationTypes: []nwc.NotificationType{
			nwc.PaymentReceived,
			nwc.PaymentSent,
			nwc.HoldInvoiceAccepted,
		},
	}
}

func handleGetInfo(walletKey signer.I) *nwc.GetInfoResult {
	fmt.Println("Handling GetInfo request")
	return &nwc.GetInfoResult{
		Alias:       "Mock Wallet",
		Color:       "#ff9900",
		Pubkey:      hex.Enc(walletKey.Pub()),
		Network:     "testnet",
		BlockHeight: 123456,
		BlockHash:   "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
		Methods: []string{
			string(nwc.GetWalletServiceInfo),
			string(nwc.GetInfo),
			string(nwc.GetBalance),
			string(nwc.GetBudget),
			string(nwc.MakeInvoice),
			string(nwc.PayInvoice),
			string(nwc.PayKeysend),
			string(nwc.LookupInvoice),
			string(nwc.ListTransactions),
			string(nwc.MakeHoldInvoice),
			string(nwc.SettleHoldInvoice),
			string(nwc.CancelHoldInvoice),
			string(nwc.SignMessage),
			string(nwc.CreateConnection),
		},
		Notifications: []string{
			string(nwc.PaymentReceived),
			string(nwc.PaymentSent),
			string(nwc.HoldInvoiceAccepted),
		},
	}
}

func handleGetBalance() *nwc.GetBalanceResult {
	fmt.Println("Handling GetBalance request")
	return &nwc.GetBalanceResult{
		Balance: 1000000, // 1,000,000 sats
	}
}

func handleGetBudget() *nwc.GetBudgetResult {
	fmt.Println("Handling GetBudget request")
	return &nwc.GetBudgetResult{
		UsedBudget:    5000,
		TotalBudget:   10000,
		RenewsAt:      int(time.Now().Add(24 * time.Hour).Unix()),
		RenewalPeriod: "daily",
	}
}

func handleMakeInvoice() *nwc.Transaction {
	fmt.Println("Handling MakeInvoice request")
	return &nwc.Transaction{
		Type:        "invoice",
		State:       "unpaid",
		Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
		Description: "Mock invoice",
		PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Amount:      1000,
		CreatedAt:   time.Now().Unix(),
		ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
	}
}

func handlePayInvoice() *nwc.PayInvoiceResult {
	fmt.Println("Handling PayInvoice request")
	return &nwc.PayInvoiceResult{
		Preimage: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		FeesPaid: 10,
	}
}

func handlePayKeysend() *nwc.PayKeysendResult {
	fmt.Println("Handling PayKeysend request")
	return &nwc.PayKeysendResult{
		Preimage: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		FeesPaid: 5,
	}
}

func handleLookupInvoice() *nwc.Transaction {
	fmt.Println("Handling LookupInvoice request")
	return &nwc.Transaction{
		Type:        "invoice",
		State:       "settled",
		Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
		Description: "Mock invoice",
		PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Preimage:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Amount:      1000,
		CreatedAt:   time.Now().Add(-1 * time.Hour).Unix(),
		ExpiresAt:   time.Now().Add(23 * time.Hour).Unix(),
	}
}

func handleListTransactions() *nwc.ListTransactionsResult {
	fmt.Println("Handling ListTransactions request")
	return &nwc.ListTransactionsResult{
		Transactions: []nwc.Transaction{
			{
				Type:        "incoming",
				State:       "settled",
				Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
				Description: "Mock incoming transaction",
				PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Preimage:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Amount:      1000,
				CreatedAt:   time.Now().Add(-24 * time.Hour).Unix(),
				ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
			},
			{
				Type:        "outgoing",
				State:       "settled",
				Invoice:     "lnbc20n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
				Description: "Mock outgoing transaction",
				PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Preimage:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Amount:      2000,
				FeesPaid:    10,
				CreatedAt:   time.Now().Add(-12 * time.Hour).Unix(),
				ExpiresAt:   time.Now().Add(36 * time.Hour).Unix(),
			},
		},
		TotalCount: 2,
	}
}

func handleMakeHoldInvoice() *nwc.Transaction {
	fmt.Println("Handling MakeHoldInvoice request")
	return &nwc.Transaction{
		Type:        "hold_invoice",
		State:       "unpaid",
		Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
		Description: "Mock hold invoice",
		PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Amount:      1000,
		CreatedAt:   time.Now().Unix(),
		ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
	}
}

func handleSignMessage() *nwc.SignMessageResult {
	fmt.Println("Handling SignMessage request")
	return &nwc.SignMessageResult{
		Message:   "Mock message",
		Signature: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}
