package nwc

import (
	"fmt"
	"time"
)

// EncryptionType represents the encryption type used for NIP-47 messages
type EncryptionType string

const (
	Nip04   EncryptionType = "nip04"
	Nip44V2 EncryptionType = "nip44_v2"
)

// AuthorizationUrlOptions represents options for creating an NWC authorization URL
type AuthorizationUrlOptions struct {
	Name              string              `json:"name,omitempty"`
	Icon              string              `json:"icon,omitempty"`
	RequestMethods    []Method            `json:"requestMethods,omitempty"`
	NotificationTypes []NotificationType  `json:"notificationTypes,omitempty"`
	ReturnTo          string              `json:"returnTo,omitempty"`
	ExpiresAt         *time.Time          `json:"expiresAt,omitempty"`
	MaxAmount         *int64              `json:"maxAmount,omitempty"`
	BudgetRenewal     BudgetRenewalPeriod `json:"budgetRenewal,omitempty"`
	Isolated          bool                `json:"isolated,omitempty"`
	Metadata          any                 `json:"metadata,omitempty"`
}

// Error is the base error type for NIP-47 errors
type Error struct {
	Message string
	Code    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s (code: %s)", e.Message, e.Code)
}

// NewError creates a new Error
func NewError(message, code string) *Error {
	return &Error{
		Message: message,
		Code:    code,
	}
}

// NetworkError represents a network error in NIP-47 operations
type NetworkError struct {
	*Error
}

// NewNetworkError creates a new NetworkError
func NewNetworkError(message, code string) *NetworkError {
	return &NetworkError{
		Error: NewError(message, code),
	}
}

// WalletError represents a wallet error in NIP-47 operations
type WalletError struct {
	*Error
}

// NewWalletError creates a new WalletError
func NewWalletError(message, code string) *WalletError {
	return &WalletError{
		Error: NewError(message, code),
	}
}

// TimeoutError represents a timeout error in NIP-47 operations
type TimeoutError struct {
	*Error
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(message, code string) *TimeoutError {
	return &TimeoutError{
		Error: NewError(message, code),
	}
}

// PublishTimeoutError represents a publish timeout error in NIP-47 operations
type PublishTimeoutError struct {
	*TimeoutError
}

// NewPublishTimeoutError creates a new PublishTimeoutError
func NewPublishTimeoutError(message, code string) *PublishTimeoutError {
	return &PublishTimeoutError{
		TimeoutError: NewTimeoutError(message, code),
	}
}

// ReplyTimeoutError represents a reply timeout error in NIP-47 operations
type ReplyTimeoutError struct {
	*TimeoutError
}

// NewReplyTimeoutError creates a new ReplyTimeoutError
func NewReplyTimeoutError(message, code string) *ReplyTimeoutError {
	return &ReplyTimeoutError{
		TimeoutError: NewTimeoutError(message, code),
	}
}

// PublishError represents a publish error in NIP-47 operations
type PublishError struct {
	*Error
}

// NewPublishError creates a new PublishError
func NewPublishError(message, code string) *PublishError {
	return &PublishError{
		Error: NewError(message, code),
	}
}

// ResponseDecodingError represents a response decoding error in NIP-47 operations
type ResponseDecodingError struct {
	*Error
}

// NewResponseDecodingError creates a new ResponseDecodingError
func NewResponseDecodingError(message, code string) *ResponseDecodingError {
	return &ResponseDecodingError{
		Error: NewError(message, code),
	}
}

// ResponseValidationError represents a response validation error in NIP-47 operations
type ResponseValidationError struct {
	*Error
}

// NewResponseValidationError creates a new ResponseValidationError
func NewResponseValidationError(message, code string) *ResponseValidationError {
	return &ResponseValidationError{
		Error: NewError(message, code),
	}
}

// UnexpectedResponseError represents an unexpected response error in NIP-47 operations
type UnexpectedResponseError struct {
	*Error
}

// NewUnexpectedResponseError creates a new UnexpectedResponseError
func NewUnexpectedResponseError(message, code string) *UnexpectedResponseError {
	return &UnexpectedResponseError{
		Error: NewError(message, code),
	}
}

// UnsupportedEncryptionError represents an unsupported encryption error in NIP-47 operations
type UnsupportedEncryptionError struct {
	*Error
}

// NewUnsupportedEncryptionError creates a new UnsupportedEncryptionError
func NewUnsupportedEncryptionError(message, code string) *UnsupportedEncryptionError {
	return &UnsupportedEncryptionError{
		Error: NewError(message, code),
	}
}

// WithDTag represents a type with a dTag field
type WithDTag struct {
	DTag string `json:"dTag"`
}

// WithOptionalId represents a type with an optional id field
type WithOptionalId struct {
	ID string `json:"id,omitempty"`
}

// Method represents a NIP-47 method
type Method []byte

// SingleMethod represents a single NIP-47 method
var (
	GetInfo           = Method("get_info")
	GetBalance        = Method("get_balance")
	GetBudget         = Method("get_budget")
	MakeInvoice       = Method("make_invoice")
	PayInvoice        = Method("pay_invoice")
	PayKeysend        = Method("pay_keysend")
	LookupInvoice     = Method("lookup_invoice")
	ListTransactions  = Method("list_transactions")
	SignMessage       = Method("sign_message")
	CreateConnection  = Method("create_connection")
	MakeHoldInvoice   = Method("make_hold_invoice")
	SettleHoldInvoice = Method("settle_hold_invoice")
	CancelHoldInvoice = Method("cancel_hold_invoice")
)

// MultiMethod represents a multi NIP-47 method
var (
	MultiPayInvoice = Method("multi_pay_invoice")
	MultiPayKeysend = Method("multi_pay_keysend")
)

// Capability represents a NIP-47 capability
type Capability []byte

var (
	Notifications = Capability("notifications")
)

// BudgetRenewalPeriod represents a budget renewal period
type BudgetRenewalPeriod string

var (
	Daily   BudgetRenewalPeriod = "daily"
	Weekly  BudgetRenewalPeriod = "weekly"
	Monthly BudgetRenewalPeriod = "monthly"
	Yearly  BudgetRenewalPeriod = "yearly"
	Never   BudgetRenewalPeriod = "never"
)

// GetInfoResponse represents a response to a get_info request
type GetInfoResponse struct {
	Alias         string             `json:"alias"`
	Color         string             `json:"color"`
	Pubkey        string             `json:"pubkey"`
	Network       string             `json:"network"`
	BlockHeight   int64              `json:"block_height"`
	BlockHash     string             `json:"block_hash"`
	Methods       []Method           `json:"methods"`
	Notifications []NotificationType `json:"notifications,omitempty"`
	Metadata      any                `json:"metadata,omitempty"`
	Lud16         string             `json:"lud16,omitempty"`
}

// GetBudgetResponse represents a response to a get_budget request
type GetBudgetResponse struct {
	UsedBudget    int64               `json:"used_budget,omitempty"`
	TotalBudget   int64               `json:"total_budget,omitempty"`
	RenewsAt      *int64              `json:"renews_at,omitempty"`
	RenewalPeriod BudgetRenewalPeriod `json:"renewal_period,omitempty"`
}

// GetBalanceResponse represents a response to a get_balance request
type GetBalanceResponse struct {
	Balance int64 `json:"balance"` // msats
}

// PayResponse represents a response to a pay request
type PayResponse struct {
	Preimage string `json:"preimage"`
	FeesPaid int64  `json:"fees_paid"`
}

// MultiPayInvoiceRequest represents a request to pay multiple invoices
type MultiPayInvoiceRequest struct {
	Invoices []PayInvoiceRequestWithID `json:"invoices"`
}

// PayInvoiceRequestWithID combines PayInvoiceRequest with WithOptionalId
type PayInvoiceRequestWithID struct {
	PayInvoiceRequest
	WithOptionalId
}

// MultiPayKeysendRequest represents a request to pay multiple keysends
type MultiPayKeysendRequest struct {
	Keysends []PayKeysendRequestWithID `json:"keysends"`
}

// PayKeysendRequestWithID combines PayKeysendRequest with WithOptionalId
type PayKeysendRequestWithID struct {
	PayKeysendRequest
	WithOptionalId
}

// MultiPayInvoiceResponse represents a response to a multi_pay_invoice request
type MultiPayInvoiceResponse struct {
	Invoices []MultiPayInvoiceResponseItem `json:"invoices"`
	Errors   []any                         `json:"errors"` // TODO: add error handling
}

// MultiPayInvoiceResponseItem represents an item in a multi_pay_invoice response
type MultiPayInvoiceResponseItem struct {
	Invoice PayInvoiceRequest `json:"invoice"`
	PayResponse
	WithDTag
}

// MultiPayKeysendResponse represents a response to a multi_pay_keysend request
type MultiPayKeysendResponse struct {
	Keysends []MultiPayKeysendResponseItem `json:"keysends"`
	Errors   []any                         `json:"errors"` // TODO: add error handling
}

// MultiPayKeysendResponseItem represents an item in a multi_pay_keysend response
type MultiPayKeysendResponseItem struct {
	Keysend PayKeysendRequest `json:"keysend"`
	PayResponse
	WithDTag
}

// ListTransactionsRequest represents a request to list transactions
type ListTransactionsRequest struct {
	From           *int64  `json:"from,omitempty"`
	Until          *int64  `json:"until,omitempty"`
	Limit          *int64  `json:"limit,omitempty"`
	Offset         *int64  `json:"offset,omitempty"`
	Unpaid         *bool   `json:"unpaid,omitempty"`
	UnpaidOutgoing *bool   `json:"unpaid_outgoing,omitempty"` // NOTE: non-NIP-47 spec compliant
	UnpaidIncoming *bool   `json:"unpaid_incoming,omitempty"` // NOTE: non-NIP-47 spec compliant
	Type           *string `json:"type,omitempty"`            // "incoming" or "outgoing"
}

// ListTransactionsResponse represents a response to a list_transactions request
type ListTransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
	TotalCount   int64         `json:"total_count"` // NOTE: non-NIP-47 spec compliant
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	Incoming TransactionType = "incoming"
	Outgoing TransactionType = "outgoing"
)

// TransactionState represents the state of a transaction
type TransactionState string

const (
	Settled TransactionState = "settled"
	Pending TransactionState = "pending"
	Failed  TransactionState = "failed"
)

// Transaction represents a transaction
type Transaction struct {
	Type            TransactionType      `json:"type"`
	State           TransactionState     `json:"state"` // NOTE: non-NIP-47 spec compliant
	Invoice         string               `json:"invoice"`
	Description     string               `json:"description"`
	DescriptionHash string               `json:"description_hash"`
	Preimage        string               `json:"preimage"`
	PaymentHash     string               `json:"payment_hash"`
	Amount          int64                `json:"amount"`
	FeesPaid        int64                `json:"fees_paid"`
	SettledAt       int64                `json:"settled_at"`
	CreatedAt       int64                `json:"created_at"`
	ExpiresAt       int64                `json:"expires_at"`
	SettleDeadline  *int64               `json:"settle_deadline,omitempty"` // NOTE: non-NIP-47 spec compliant
	Metadata        *TransactionMetadata `json:"metadata,omitempty"`
}

// TransactionMetadata represents metadata for a transaction
type TransactionMetadata struct {
	Comment       string         `json:"comment,omitempty"`        // LUD-12
	PayerData     *PayerData     `json:"payer_data,omitempty"`     // LUD-18
	RecipientData *RecipientData `json:"recipient_data,omitempty"` // LUD-18
	Nostr         *NostrData     `json:"nostr,omitempty"`          // NIP-57
	ExtraData     map[string]any `json:"-"`                        // For additional fields
}

// PayerData represents payer data for a transaction
type PayerData struct {
	Email  string `json:"email,omitempty"`
	Name   string `json:"name,omitempty"`
	Pubkey string `json:"pubkey,omitempty"`
}

// RecipientData represents recipient data for a transaction
type RecipientData struct {
	Identifier string `json:"identifier,omitempty"`
}

// NostrData represents Nostr data for a transaction
type NostrData struct {
	Pubkey string     `json:"pubkey"`
	Tags   [][]string `json:"tags"`
}

// NotificationType represents a notification type
type NotificationType []byte

var (
	PaymentReceived     NotificationType = []byte("payment_received")
	PaymentSent         NotificationType = []byte("payment_sent")
	HoldInvoiceAccepted NotificationType = []byte("hold_invoice_accepted")
)

// Notification represents a notification
type Notification struct {
	NotificationType NotificationType `json:"notification_type"`
	Notification     Transaction      `json:"notification"`
}

// PayInvoiceRequest represents a request to pay an invoice
type PayInvoiceRequest struct {
	Invoice  string               `json:"invoice"`
	Metadata *TransactionMetadata `json:"metadata,omitempty"`
	Amount   *int64               `json:"amount,omitempty"` // msats
}

// PayKeysendRequest represents a request to pay a keysend
type PayKeysendRequest struct {
	Amount     int64       `json:"amount"` // msats
	Pubkey     string      `json:"pubkey"`
	Preimage   string      `json:"preimage,omitempty"`
	TlvRecords []TLVRecord `json:"tlv_records,omitempty"`
}

// TLVRecord represents a TLV record
type TLVRecord struct {
	Type  int64  `json:"type"`
	Value string `json:"value"`
}

// MakeInvoiceRequest represents a request to make an invoice
type MakeInvoiceRequest struct {
	Amount          int64                `json:"amount"` // msats
	Description     string               `json:"description,omitempty"`
	DescriptionHash string               `json:"description_hash,omitempty"`
	Expiry          *int64               `json:"expiry,omitempty"` // in seconds
	Metadata        *TransactionMetadata `json:"metadata,omitempty"`
}

// MakeHoldInvoiceRequest represents a request to make a hold invoice
type MakeHoldInvoiceRequest struct {
	MakeInvoiceRequest
	PaymentHash string `json:"payment_hash"`
}

// SettleHoldInvoiceRequest represents a request to settle a hold invoice
type SettleHoldInvoiceRequest struct {
	Preimage string `json:"preimage"`
}

// SettleHoldInvoiceResponse represents a response to a settle_hold_invoice request
type SettleHoldInvoiceResponse struct{}

// CancelHoldInvoiceRequest represents a request to cancel a hold invoice
type CancelHoldInvoiceRequest struct {
	PaymentHash string `json:"payment_hash"`
}

// CancelHoldInvoiceResponse represents a response to a cancel_hold_invoice request
type CancelHoldInvoiceResponse struct{}

// LookupInvoiceRequest represents a request to lookup an invoice
type LookupInvoiceRequest struct {
	PaymentHash string `json:"payment_hash,omitempty"`
	Invoice     string `json:"invoice,omitempty"`
}

// SignMessageRequest represents a request to sign a message
type SignMessageRequest struct {
	Message string `json:"message"`
}

// CreateConnectionRequest represents a request to create a connection
type CreateConnectionRequest struct {
	Pubkey            string               `json:"pubkey"`
	Name              string               `json:"name"`
	RequestMethods    []Method             `json:"request_methods"`
	NotificationTypes []NotificationType   `json:"notification_types,omitempty"`
	MaxAmount         *int64               `json:"max_amount,omitempty"`
	BudgetRenewal     *BudgetRenewalPeriod `json:"budget_renewal,omitempty"`
	ExpiresAt         *int64               `json:"expires_at,omitempty"`
	Isolated          *bool                `json:"isolated,omitempty"`
	Metadata          any                  `json:"metadata,omitempty"`
}

// CreateConnectionResponse represents a response to a create_connection request
type CreateConnectionResponse struct {
	WalletPubkey string `json:"wallet_pubkey"`
}

// SignMessageResponse represents a response to a sign_message request
type SignMessageResponse struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

// TimeoutValues represents timeout values for NIP-47 requests
type TimeoutValues struct {
	ReplyTimeout   *int64 `json:"replyTimeout,omitempty"`
	PublishTimeout *int64 `json:"publishTimeout,omitempty"`
}
