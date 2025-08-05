package nwc

// Capability represents a NIP-47 method
type Capability []byte

var (
	GetInfo           = Capability("get_info")
	GetBalance        = Capability("get_balance")
	GetBudget         = Capability("get_budget")
	MakeInvoice       = Capability("make_invoice")
	PayInvoice        = Capability("pay_invoice")
	PayKeysend        = Capability("pay_keysend")
	LookupInvoice     = Capability("lookup_invoice")
	ListTransactions  = Capability("list_transactions")
	SignMessage       = Capability("sign_message")
	CreateConnection  = Capability("create_connection")
	MakeHoldInvoice   = Capability("make_hold_invoice")
	SettleHoldInvoice = Capability("settle_hold_invoice")
	CancelHoldInvoice = Capability("cancel_hold_invoice")
	MultiPayInvoice   = Capability("multi_pay_invoice")
	MultiPayKeysend   = Capability("multi_pay_keysend")
)

// EncryptionType represents the encryption type used for NIP-47 messages
type EncryptionType []byte

var (
	Nip04   = EncryptionType("nip04")
	Nip44V2 = EncryptionType("nip44_v2")
)

type NotificationType []byte

var (
	PaymentReceived = NotificationType("payment_received")
	PaymentSent     = NotificationType("payment_sent")
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
	BlockHeight   uint     `json:"block_height"`
	BlockHash     string   `json:"block_hash"`
	Methods       []string `json:"methods"`
	Notifications []string `json:"notifications"`
}

type MakeInvoiceParams struct {
	Amount          uint64  `json:"amount"`
	Expiry          *uint32 `json:"expiry"`
	Description     string  `json:"description"`
	DescriptionHash string  `json:"description_hash"`
	Metadata        any     `json:"metadata"`
}

type PayInvoiceParams struct {
	Invoice  string  `json:"invoice"`
	Amount   *uint64 `json:"amount"`
	Metadata any     `json:"metadata"`
}

type LookupInvoiceParams struct {
	PaymentHash string `json:"payment_hash"`
	Invoice     string `json:"invoice"`
}

type ListTransactionsParams struct {
	From           uint64 `json:"from"`
	To             uint64 `json:"to"`
	Limit          uint16 `json:"limit"`
	Offset         uint32 `json:"offset"`
	Unpaid         bool   `json:"unpaid"`
	UnpaidOutgoing bool   `json:"unpaid_outgoing"`
	UnpaidIncoming bool   `json:"unpaid_incoming"`
	Type           string `json:"type"`
}

type GetBalanceResult struct {
	Balance uint64 `json:"balance"`
}

type PayInvoiceResult struct {
	Preimage string `json:"preimage"`
	FeesPaid uint64 `json:"fees_paid"`
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
	CreatedAt       uint64  `json:"created_at"`
	ExpiresAt       uint64  `json:"expires_at"`
	SettledAt       *uint64 `json:"settled_at"`
	Metadata        any     `json:"metadata"`
}
