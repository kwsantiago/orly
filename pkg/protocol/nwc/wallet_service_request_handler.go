package nwc

// WalletServiceRequestHandlerError represents an error from a wallet service request handler
type WalletServiceRequestHandlerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WalletServiceResponse represents a response from a wallet service request handler
type WalletServiceResponse struct {
	Result interface{}                       `json:"result,omitempty"`
	Error  *WalletServiceRequestHandlerError `json:"error,omitempty"`
}

// WalletServiceRequestHandler is an interface for handling wallet service requests
type WalletServiceRequestHandler interface {
	// GetInfo returns information about the wallet
	GetInfo() (*WalletServiceResponse, error)

	// MakeInvoice creates a new invoice
	MakeInvoice(request *MakeInvoiceRequest) (
		*WalletServiceResponse, error,
	)

	// PayInvoice pays an invoice
	PayInvoice(request *PayInvoiceRequest) (
		*WalletServiceResponse, error,
	)

	// PayKeysend sends a keysend payment
	PayKeysend(request *PayKeysendRequest) (*WalletServiceResponse, error)

	// GetBalance returns the wallet balance
	GetBalance() (*WalletServiceResponse, error)

	// LookupInvoice looks up an invoice
	LookupInvoice(request *LookupInvoiceRequest) (
		*WalletServiceResponse, error,
	)

	// ListTransactions lists transactions
	ListTransactions(request *ListTransactionsRequest) (
		*WalletServiceResponse, error,
	)

	// SignMessage signs a message
	SignMessage(request *SignMessageRequest) (
		*WalletServiceResponse, error,
	)
}
