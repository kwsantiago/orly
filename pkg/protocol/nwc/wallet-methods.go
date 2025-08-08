package nwc

import (
	"encoding/json"
	"fmt"

	"orly.dev/pkg/utils/context"
)

// handleGetWalletServiceInfo handles the GetWalletServiceInfo method.
func (ws *WalletService) handleGetWalletServiceInfo(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Empty stub implementation
	return &WalletServiceInfo{}, nil
}

// handleCancelHoldInvoice handles the CancelHoldInvoice method.
func (ws *WalletService) handleCancelHoldInvoice(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p CancelHoldInvoiceParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return nil, nil
}

// handleCreateConnection handles the CreateConnection method.
func (ws *WalletService) handleCreateConnection(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p CreateConnectionParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return nil, nil
}

// handleGetBalance handles the GetBalance method.
func (ws *WalletService) handleGetBalance(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Empty stub implementation
	return &GetBalanceResult{}, nil
}

// handleGetBudget handles the GetBudget method.
func (ws *WalletService) handleGetBudget(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Empty stub implementation
	return &GetBudgetResult{}, nil
}

// handleGetInfo handles the GetInfo method.
func (ws *WalletService) handleGetInfo(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Empty stub implementation
	return &GetInfoResult{}, nil
}

// handleListTransactions handles the ListTransactions method.
func (ws *WalletService) handleListTransactions(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p ListTransactionsParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return &ListTransactionsResult{}, nil
}

// handleLookupInvoice handles the LookupInvoice method.
func (ws *WalletService) handleLookupInvoice(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p LookupInvoiceParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return &LookupInvoiceResult{}, nil
}

// handleMakeHoldInvoice handles the MakeHoldInvoice method.
func (ws *WalletService) handleMakeHoldInvoice(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p MakeHoldInvoiceParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return &MakeInvoiceResult{}, nil
}

// handleMakeInvoice handles the MakeInvoice method.
func (ws *WalletService) handleMakeInvoice(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p MakeInvoiceParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return &MakeInvoiceResult{}, nil
}

// handlePayKeysend handles the PayKeysend method.
func (ws *WalletService) handlePayKeysend(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p PayKeysendParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return &PayKeysendResult{}, nil
}

// handlePayInvoice handles the PayInvoice method.
func (ws *WalletService) handlePayInvoice(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p PayInvoiceParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return &PayInvoiceResult{}, nil
}

// handleSettleHoldInvoice handles the SettleHoldInvoice method.
func (ws *WalletService) handleSettleHoldInvoice(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p SettleHoldInvoiceParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return nil, nil
}

// handleSignMessage handles the SignMessage method.
func (ws *WalletService) handleSignMessage(c context.T, params json.RawMessage) (result interface{}, err error) {
	// Parse parameters
	var p SignMessageParams
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, &ResponseError{
			Code:    "invalid_params",
			Message: fmt.Sprintf("failed to parse parameters: %v", err),
		}
	}

	// Empty stub implementation
	return &SignMessageResult{}, nil
}