package nwc

// Capability represents a NIP-47 method
type Capability []byte

var (
	CancelHoldInvoice    = Capability("cancel_hold_invoice")
	CreateConnection     = Capability("create_connection")
	GetBalance           = Capability("get_balance")
	GetBudget            = Capability("get_budget")
	GetInfo              = Capability("get_info")
	GetWalletServiceInfo = Capability("get_wallet_service_info")
	ListTransactions     = Capability("list_transactions")
	LookupInvoice        = Capability("lookup_invoice")
	MakeHoldInvoice      = Capability("make_hold_invoice")
	MakeInvoice          = Capability("make_invoice")
	MultiPayInvoice      = Capability("multi_pay_invoice")
	MultiPayKeysend      = Capability("multi_pay_keysend")
	PayInvoice           = Capability("pay_invoice")
	PayKeysend           = Capability("pay_keysend")
	SettleHoldInvoice    = Capability("settle_hold_invoice")
	SignMessage          = Capability("sign_message")
)

// EncryptionType represents the encryption type used for NIP-47 messages
type EncryptionType []byte

var (
	EncryptionTag = []byte("encryption")
	Nip04         = EncryptionType("nip04")
	Nip44V2       = EncryptionType("nip44_v2")
)

type NotificationType []byte

var (
	NotificationTag     = []byte("notification")
	PaymentReceived     = NotificationType("payment_received")
	PaymentSent         = NotificationType("payment_sent")
	HoldInvoiceAccepted = NotificationType("hold_invoice_accepted")
)

type WalletServiceInfo struct {
	EncryptionTypes   []EncryptionType
	Capabilities      []Capability
	NotificationTypes []NotificationType
}

type GetInfoResult struct {
	Alias         string   `json:"alias"`
	Color         string   `json:"color"`
	Pubkey        string   `json:"pubkey"`
	Network       string   `json:"network"`
	BlockHeight   uint64   `json:"block_height"`
	BlockHash     string   `json:"block_hash"`
	Methods       []string `json:"methods"`
	Notifications []string `json:"notifications,omitempty"`
	Metadata      any      `json:"metadata,omitempty"`
	LUD16         string   `json:"lud16,omitempty"`
}

type GetBudgetResult struct {
	UsedBudget    int    `json:"used_budget,omitempty"`
	TotalBudget   int    `json:"total_budget,omitempty"`
	RenewsAt      int    `json:"renews_at,omitempty"`
	RenewalPeriod string `json:"renewal_period,omitempty"`
}

type GetBalanceResult struct {
	Balance uint64 `json:"balance"`
}

type MakeInvoiceParams struct {
	Amount          uint64 `json:"amount"`
	Description     string `json:"description,omitempty"`
	DescriptionHash string `json:"description_hash,omitempty"`
	Expiry          *int64 `json:"expiry,omitempty"`
	Metadata        any    `json:"metadata,omitempty"`
}

type MakeHoldInvoiceParams struct {
	Amount          uint64 `json:"amount"`
	PaymentHash     string `json:"payment_hash"`
	Description     string `json:"description,omitempty"`
	DescriptionHash string `json:"description_hash,omitempty"`
	Expiry          *int64 `json:"expiry,omitempty"`
	Metadata        any    `json:"metadata,omitempty"`
}

type SettleHoldInvoiceParams struct {
	Preimage string `json:"preimage"`
}

type CancelHoldInvoiceParams struct {
	PaymentHash string `json:"payment_hash"`
}

type PayInvoicePayerData struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Pubkey string `json:"pubkey"`
}

type PayInvoiceMetadata struct {
	Comment   *string              `json:"comment"`
	PayerData *PayInvoicePayerData `json:"payer_data"`
	Other     any
}

type PayInvoiceParams struct {
	Invoice  string              `json:"invoice"`
	Amount   *uint64             `json:"amount,omitempty"`
	Metadata *PayInvoiceMetadata `json:"metadata,omitempty"`
}

type PayInvoiceResult struct {
	Preimage string `json:"preimage"`
	FeesPaid uint64 `json:"fees_paid"`
}

type PayKeysendTLVRecord struct {
	Type  uint32 `json:"type"`
	Value string `json:"value"`
}

type PayKeysendParams struct {
	Amount     uint64                `json:"amount"`
	Pubkey     string                `json:"pubkey"`
	Preimage   *string               `json:"preimage,omitempty"`
	TLVRecords []PayKeysendTLVRecord `json:"tlv_records,omitempty"`
}

type PayKeysendResult = PayInvoiceResult

type LookupInvoiceParams struct {
	PaymentHash *string `json:"payment_hash,omitempty"`
	Invoice     *string `json:"invoice,omitempty"`
}

type ListTransactionsParams struct {
	From           *int64  `json:"from,omitempty"`
	Until          *int64  `json:"until,omitempty"`
	Limit          *uint16 `json:"limit,omitempty"`
	Offset         *uint32 `json:"offset,omitempty"`
	Unpaid         *bool   `json:"unpaid,omitempty"`
	UnpaidOutgoing *bool   `json:"unpaid_outgoing,omitempty"`
	UnpaidIncoming *bool   `json:"unpaid_incoming,omitempty"`
	Type           *string `json:"type,omitempty"`
}

type MakeInvoiceResult = Transaction
type LookupInvoiceResult = Transaction
type ListTransactionsResult struct {
	Transactions []Transaction `json:"transactions"`
	TotalCount   uint32        `json:"total_count"`
}

type Transaction struct {
	Type            string  `json:"type"`
	State           string  `json:"state"`
	Invoice         string  `json:"invoice"`
	Description     string  `json:"description"`
	DescriptionHash string  `json:"description_hash"`
	Preimage        string  `json:"preimage"`
	PaymentHash     string  `json:"payment_hash"`
	Amount          uint64  `json:"amount"`
	FeesPaid        uint64  `json:"fees_paid"`
	CreatedAt       int64   `json:"created_at"`
	ExpiresAt       int64   `json:"expires_at"`
	SettledDeadline *uint64 `json:"settled_deadline,omitempty"`
	Metadata        any     `json:"metadata,omitempty"`
}

type SignMessageParams struct {
	Message string `json:"message"`
}

type SignMessageResult struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

type CreateConnectionParams struct {
	Pubkey            string   `json:"pubkey"`
	Name              string   `json:"name"`
	RequestMethods    []string `json:"request_methods"`
	NotificationTypes []string `json:"notification_types"`
	MaxAmount         *uint64  `json:"max_amount,omitempty"`
	BudgetRenewal     *string  `json:"budget_renewal,omitempty"`
	ExpiresAt         *int64   `json:"expires_at,omitempty"`
}
