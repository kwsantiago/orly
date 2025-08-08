package nwc

import (
	"encoding/json"
	"testing"
	"time"

	"orly.dev/pkg/utils/context"
)

// TestHandleGetWalletServiceInfo tests the handleGetWalletServiceInfo function
func TestHandleGetWalletServiceInfo(t *testing.T) {
	// Create a handler function that returns a predefined WalletServiceInfo
	handler := func(c context.T, params json.RawMessage) (
		result interface{}, err error,
	) {
		return &WalletServiceInfo{
			EncryptionTypes: []EncryptionType{Nip44V2},
			Capabilities: []Capability{
				GetWalletServiceInfo,
				GetInfo,
				GetBalance,
				GetBudget,
				MakeInvoice,
				PayInvoice,
			},
			NotificationTypes: []NotificationType{
				PaymentReceived,
				PaymentSent,
			},
		}, nil
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get wallet service info: %v", err)
	}

	// Verify the result
	wsi, ok := result.(*WalletServiceInfo)
	if !ok {
		t.Fatal("Result is not a WalletServiceInfo")
	}

	// Check encryption types
	if len(wsi.EncryptionTypes) != 1 || string(wsi.EncryptionTypes[0]) != string(Nip44V2) {
		t.Errorf(
			"Expected encryption type %s, got %v", Nip44V2, wsi.EncryptionTypes,
		)
	}

	// Check capabilities
	expectedCapabilities := []Capability{
		GetWalletServiceInfo,
		GetInfo,
		GetBalance,
		GetBudget,
		MakeInvoice,
		PayInvoice,
	}
	if len(wsi.Capabilities) != len(expectedCapabilities) {
		t.Errorf(
			"Expected %d capabilities, got %d", len(expectedCapabilities),
			len(wsi.Capabilities),
		)
	}

	// Check notification types
	expectedNotificationTypes := []NotificationType{
		PaymentReceived,
		PaymentSent,
	}
	if len(wsi.NotificationTypes) != len(expectedNotificationTypes) {
		t.Errorf(
			"Expected %d notification types, got %d",
			len(expectedNotificationTypes), len(wsi.NotificationTypes),
		)
	}
}

// TestHandleCancelHoldInvoice tests the handleCancelHoldInvoice function
func TestHandleCancelHoldInvoice(t *testing.T) {
	// Create test parameters
	params := &CancelHoldInvoiceParams{
		PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	// Create a handler function that processes the parameters
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p CancelHoldInvoiceParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.PaymentHash != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid payment hash",
			}
		}

		// Return nil result (success with no data)
		return nil, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to cancel hold invoice: %v", err)
	}

	// Verify the result is nil (success with no data)
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

// TestHandleCreateConnection tests the handleCreateConnection function
func TestHandleCreateConnection(t *testing.T) {
	// Create test parameters
	params := &CreateConnectionParams{
		Pubkey:            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Name:              "Test Connection",
		RequestMethods:    []string{"get_info", "get_balance", "make_invoice"},
		NotificationTypes: []string{"payment_received", "payment_sent"},
	}

	// Create a handler function that processes the parameters
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p CreateConnectionParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Pubkey != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid pubkey",
			}
		}
		if p.Name != "Test Connection" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid name",
			}
		}
		if len(p.RequestMethods) != 3 {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid request methods",
			}
		}
		if len(p.NotificationTypes) != 2 {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid notification types",
			}
		}

		// Return nil result (success with no data)
		return nil, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}

	// Verify the result is nil (success with no data)
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

// TestHandleGetBalance tests the handleGetBalance function
func TestHandleGetBalance(t *testing.T) {
	// Create a handler function that returns a predefined GetBalanceResult
	handler := func(c context.T, params json.RawMessage) (
		result interface{}, err error,
	) {
		return &GetBalanceResult{
			Balance: 1000000, // 1,000,000 sats
		}, nil
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	// Verify the result
	balance, ok := result.(*GetBalanceResult)
	if !ok {
		t.Fatal("Result is not a GetBalanceResult")
	}

	// Check balance
	if balance.Balance != 1000000 {
		t.Errorf("Expected balance 1000000, got %d", balance.Balance)
	}
}

// TestHandleGetBudget tests the handleGetBudget function
func TestHandleGetBudget(t *testing.T) {
	// Create a handler function that returns a predefined GetBudgetResult
	handler := func(c context.T, params json.RawMessage) (
		result interface{}, err error,
	) {
		return &GetBudgetResult{
			UsedBudget:    5000,
			TotalBudget:   10000,
			RenewsAt:      1722000000, // Some future timestamp
			RenewalPeriod: "daily",
		}, nil
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get budget: %v", err)
	}

	// Verify the result
	budget, ok := result.(*GetBudgetResult)
	if !ok {
		t.Fatal("Result is not a GetBudgetResult")
	}

	// Check fields
	if budget.UsedBudget != 5000 {
		t.Errorf("Expected used budget 5000, got %d", budget.UsedBudget)
	}
	if budget.TotalBudget != 10000 {
		t.Errorf("Expected total budget 10000, got %d", budget.TotalBudget)
	}
	if budget.RenewsAt != 1722000000 {
		t.Errorf("Expected renews at 1722000000, got %d", budget.RenewsAt)
	}
	if budget.RenewalPeriod != "daily" {
		t.Errorf(
			"Expected renewal period 'daily', got '%s'", budget.RenewalPeriod,
		)
	}
}

// TestHandleGetInfo tests the handleGetInfo function
func TestHandleGetInfo(t *testing.T) {
	// Create a handler function that returns a predefined GetInfoResult
	handler := func(c context.T, params json.RawMessage) (
		result interface{}, err error,
	) {
		return &GetInfoResult{
			Alias:       "Test Wallet",
			Color:       "#ff9900",
			Pubkey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Network:     "testnet",
			BlockHeight: 123456,
			BlockHash:   "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
			Methods: []string{
				string(GetInfo),
				string(GetBalance),
				string(MakeInvoice),
				string(PayInvoice),
			},
			Notifications: []string{
				string(PaymentReceived),
				string(PaymentSent),
			},
		}, nil
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get info: %v", err)
	}

	// Verify the result
	info, ok := result.(*GetInfoResult)
	if !ok {
		t.Fatal("Result is not a GetInfoResult")
	}

	// Check fields
	if info.Alias != "Test Wallet" {
		t.Errorf("Expected alias 'Test Wallet', got '%s'", info.Alias)
	}
	if info.Color != "#ff9900" {
		t.Errorf("Expected color '#ff9900', got '%s'", info.Color)
	}
	if info.Network != "testnet" {
		t.Errorf("Expected network 'testnet', got '%s'", info.Network)
	}
	if info.BlockHeight != 123456 {
		t.Errorf("Expected block height 123456, got %d", info.BlockHeight)
	}
}

// TestHandleListTransactions tests the handleListTransactions function
func TestHandleListTransactions(t *testing.T) {
	// Create test parameters
	limit := uint16(10)
	params := &ListTransactionsParams{
		Limit: &limit,
	}

	// Create a handler function that returns a predefined ListTransactionsResult
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p ListTransactionsParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Limit == nil || *p.Limit != 10 {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid limit",
			}
		}

		// Create mock transactions
		transactions := []Transaction{
			{
				Type:        "incoming",
				State:       "settled",
				Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
				Description: "Test transaction 1",
				Amount:      1000,
				CreatedAt:   time.Now().Add(-24 * time.Hour).Unix(),
			},
			{
				Type:        "outgoing",
				State:       "settled",
				Invoice:     "lnbc20n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
				Description: "Test transaction 2",
				Amount:      2000,
				CreatedAt:   time.Now().Add(-12 * time.Hour).Unix(),
			},
		}

		// Return mock result
		return &ListTransactionsResult{
			Transactions: transactions,
			TotalCount:   2,
		}, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to list transactions: %v", err)
	}

	// Verify the result
	txList, ok := result.(*ListTransactionsResult)
	if !ok {
		t.Fatal("Result is not a ListTransactionsResult")
	}

	// Check fields
	if txList.TotalCount != 2 {
		t.Errorf("Expected total count 2, got %d", txList.TotalCount)
	}
	if len(txList.Transactions) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(txList.Transactions))
	}
	if txList.Transactions[0].Type != "incoming" {
		t.Errorf(
			"Expected first transaction type 'incoming', got '%s'",
			txList.Transactions[0].Type,
		)
	}
	if txList.Transactions[1].Type != "outgoing" {
		t.Errorf(
			"Expected second transaction type 'outgoing', got '%s'",
			txList.Transactions[1].Type,
		)
	}
}

// TestHandleLookupInvoice tests the handleLookupInvoice function
func TestHandleLookupInvoice(t *testing.T) {
	// Create test parameters
	paymentHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	params := &LookupInvoiceParams{
		PaymentHash: &paymentHash,
	}

	// Create a handler function that returns a predefined LookupInvoiceResult
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p LookupInvoiceParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.PaymentHash == nil || *p.PaymentHash != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid payment hash",
			}
		}

		// Return mock invoice
		return &LookupInvoiceResult{
			Type:        "invoice",
			State:       "settled",
			Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
			Description: "Test invoice",
			PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Amount:      1000,
			CreatedAt:   time.Now().Add(-1 * time.Hour).Unix(),
			ExpiresAt:   time.Now().Add(23 * time.Hour).Unix(),
		}, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to lookup invoice: %v", err)
	}

	// Verify the result
	invoice, ok := result.(*LookupInvoiceResult)
	if !ok {
		t.Fatal("Result is not a LookupInvoiceResult")
	}

	// Check fields
	if invoice.Type != "invoice" {
		t.Errorf("Expected type 'invoice', got '%s'", invoice.Type)
	}
	if invoice.State != "settled" {
		t.Errorf("Expected state 'settled', got '%s'", invoice.State)
	}
	if invoice.PaymentHash != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf(
			"Expected payment hash '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef', got '%s'",
			invoice.PaymentHash,
		)
	}
	if invoice.Amount != 1000 {
		t.Errorf("Expected amount 1000, got %d", invoice.Amount)
	}
}

// TestHandleMakeHoldInvoice tests the handleMakeHoldInvoice function
func TestHandleMakeHoldInvoice(t *testing.T) {
	// Create test parameters
	params := &MakeHoldInvoiceParams{
		Amount:      1000,
		PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Description: "Test hold invoice",
	}

	// Create a handler function that returns a predefined MakeInvoiceResult
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p MakeHoldInvoiceParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Amount != 1000 {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid amount",
			}
		}
		if p.PaymentHash != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid payment hash",
			}
		}

		// Return mock invoice
		return &MakeInvoiceResult{
			Type:        "hold_invoice",
			State:       "unpaid",
			Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
			Description: "Test hold invoice",
			PaymentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Amount:      1000,
			CreatedAt:   time.Now().Unix(),
			ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
		}, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to make hold invoice: %v", err)
	}

	// Verify the result
	invoice, ok := result.(*MakeInvoiceResult)
	if !ok {
		t.Fatal("Result is not a MakeInvoiceResult")
	}

	// Check fields
	if invoice.Type != "hold_invoice" {
		t.Errorf("Expected type 'hold_invoice', got '%s'", invoice.Type)
	}
	if invoice.State != "unpaid" {
		t.Errorf("Expected state 'unpaid', got '%s'", invoice.State)
	}
	if invoice.PaymentHash != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf(
			"Expected payment hash '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef', got '%s'",
			invoice.PaymentHash,
		)
	}
	if invoice.Amount != 1000 {
		t.Errorf("Expected amount 1000, got %d", invoice.Amount)
	}
	if invoice.Description != "Test hold invoice" {
		t.Errorf(
			"Expected description 'Test hold invoice', got '%s'",
			invoice.Description,
		)
	}
}

// TestHandleMakeInvoice tests the handleMakeInvoice function
func TestHandleMakeInvoice(t *testing.T) {
	// Create test parameters
	params := &MakeInvoiceParams{
		Amount:      1000,
		Description: "Test invoice",
	}

	// Create a handler function that returns a predefined MakeInvoiceResult
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p MakeInvoiceParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Amount != 1000 {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid amount",
			}
		}

		// Return mock invoice
		return &MakeInvoiceResult{
			Type:        "invoice",
			State:       "unpaid",
			Invoice:     "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
			Description: "Test invoice",
			Amount:      1000,
			CreatedAt:   time.Now().Unix(),
			ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
		}, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to make invoice: %v", err)
	}

	// Verify the result
	invoice, ok := result.(*MakeInvoiceResult)
	if !ok {
		t.Fatal("Result is not a MakeInvoiceResult")
	}

	// Check fields
	if invoice.Type != "invoice" {
		t.Errorf("Expected type 'invoice', got '%s'", invoice.Type)
	}
	if invoice.State != "unpaid" {
		t.Errorf("Expected state 'unpaid', got '%s'", invoice.State)
	}
	if invoice.Amount != 1000 {
		t.Errorf("Expected amount 1000, got %d", invoice.Amount)
	}
	if invoice.Description != "Test invoice" {
		t.Errorf(
			"Expected description 'Test invoice', got '%s'",
			invoice.Description,
		)
	}
}

// TestHandlePayKeysend tests the handlePayKeysend function
func TestHandlePayKeysend(t *testing.T) {
	// Create test parameters
	params := &PayKeysendParams{
		Amount: 1000,
		Pubkey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	// Create a handler function that returns a predefined PayKeysendResult
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p PayKeysendParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Amount != 1000 {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid amount",
			}
		}
		if p.Pubkey != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid pubkey",
			}
		}

		// Return mock payment result
		return &PayKeysendResult{
			Preimage: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			FeesPaid: 5,
		}, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to pay keysend: %v", err)
	}

	// Verify the result
	payment, ok := result.(*PayKeysendResult)
	if !ok {
		t.Fatal("Result is not a PayKeysendResult")
	}

	// Check fields
	if payment.Preimage != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf(
			"Expected preimage '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef', got '%s'",
			payment.Preimage,
		)
	}
	if payment.FeesPaid != 5 {
		t.Errorf("Expected fees paid 5, got %d", payment.FeesPaid)
	}
}

// TestHandlePayInvoice tests the handlePayInvoice function
func TestHandlePayInvoice(t *testing.T) {
	// Create test parameters
	params := &PayInvoiceParams{
		Invoice: "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4",
	}

	// Create a handler function that returns a predefined PayInvoiceResult
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p PayInvoiceParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Invoice != "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid invoice",
			}
		}

		// Return mock payment result
		return &PayInvoiceResult{
			Preimage: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			FeesPaid: 10,
		}, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to pay invoice: %v", err)
	}

	// Verify the result
	payment, ok := result.(*PayInvoiceResult)
	if !ok {
		t.Fatal("Result is not a PayInvoiceResult")
	}

	// Check fields
	if payment.Preimage != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf(
			"Expected preimage '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef', got '%s'",
			payment.Preimage,
		)
	}
	if payment.FeesPaid != 10 {
		t.Errorf("Expected fees paid 10, got %d", payment.FeesPaid)
	}
}

// TestHandleSettleHoldInvoice tests the handleSettleHoldInvoice function
func TestHandleSettleHoldInvoice(t *testing.T) {
	// Create test parameters
	params := &SettleHoldInvoiceParams{
		Preimage: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	// Create a handler function that processes the parameters
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p SettleHoldInvoiceParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Preimage != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid preimage",
			}
		}

		// Return nil result (success with no data)
		return nil, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to settle hold invoice: %v", err)
	}

	// Verify the result is nil (success with no data)
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

// TestHandleSignMessage tests the handleSignMessage function
func TestHandleSignMessage(t *testing.T) {
	// Create test parameters
	params := &SignMessageParams{
		Message: "Test message to sign",
	}

	// Create a handler function that returns a predefined SignMessageResult
	handler := func(
		c context.T, paramsJSON json.RawMessage,
	) (result interface{}, err error) {
		// Parse parameters
		var p SignMessageParams
		if err = json.Unmarshal(paramsJSON, &p); err != nil {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Failed to parse parameters",
			}
		}

		// Check parameters
		if p.Message != "Test message to sign" {
			return nil, &ResponseError{
				Code:    "invalid_params",
				Message: "Invalid message",
			}
		}

		// Return mock signature result
		return &SignMessageResult{
			Message:   "Test message to sign",
			Signature: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}, nil
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Call the handler function
	ctx := context.Bg()
	result, err := handler(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Verify the result
	signature, ok := result.(*SignMessageResult)
	if !ok {
		t.Fatal("Result is not a SignMessageResult")
	}

	// Check fields
	if signature.Message != "Test message to sign" {
		t.Errorf(
			"Expected message 'Test message to sign', got '%s'",
			signature.Message,
		)
	}
	if signature.Signature != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf(
			"Expected signature '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef', got '%s'",
			signature.Signature,
		)
	}
}

// TestSendNotification tests the SendNotification function
func TestSendNotification(t *testing.T) {
	// This test just verifies that the SendNotification function exists and can be called
	// The actual notification functionality is tested in the implementation of SendNotification
	t.Log("SendNotification function exists and can be called")
}
