package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/protocol/nwc"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/interrupt"
)

func printUsage() {
	fmt.Println("Usage: walletcli \"<NWC connection URL>\" <method> [<args...>]")
	fmt.Println("\nAvailable methods:")
	fmt.Println("  get_wallet_service_info - Get wallet service information")
	fmt.Println("  get_info                - Get wallet information")
	fmt.Println("  get_balance             - Get wallet balance")
	fmt.Println("  get_budget              - Get wallet budget")
	fmt.Println("  make_invoice            - Create an invoice")
	fmt.Println("                            Args: <amount> [<description>] [<description_hash>] [<expiry>]")
	fmt.Println("  pay_invoice             - Pay an invoice")
	fmt.Println("                            Args: <invoice> [<amount>] [<comment>]")
	fmt.Println("  pay_keysend             - Pay to a node using keysend")
	fmt.Println("                            Args: <pubkey> <amount> [<preimage>] [<tlv_type> <tlv_value>...]")
	fmt.Println("  lookup_invoice          - Look up an invoice")
	fmt.Println("                            Args: <payment_hash or invoice>")
	fmt.Println("  list_transactions       - List transactions")
	fmt.Println("                            Args: [<limit>] [<offset>] [<from>] [<until>]")
	fmt.Println("  make_hold_invoice       - Create a hold invoice")
	fmt.Println("                            Args: <amount> <payment_hash> [<description>] [<description_hash>] [<expiry>]")
	fmt.Println("  settle_hold_invoice     - Settle a hold invoice")
	fmt.Println("                            Args: <preimage>")
	fmt.Println("  cancel_hold_invoice     - Cancel a hold invoice")
	fmt.Println("                            Args: <payment_hash>")
	fmt.Println("  sign_message            - Sign a message")
	fmt.Println("                            Args: <message>")
	fmt.Println("  create_connection       - Create a connection")
	fmt.Println("                            Args: <pubkey> <name> <methods> [<notification_types>] [<max_amount>] [<budget_renewal>] [<expires_at>]")
	fmt.Println("  subscribe               - Subscribe to payment_received, payment_sent and hold_invoice_accepted notifications visible in the scope of the connection")
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}
	connectionURL := os.Args[1]
	method := os.Args[2]
	args := os.Args[3:]
	// Create context
	// c, cancel := context.Cancel(context.Bg())
	c := context.Bg()
	// defer cancel()
	// Create NWC client
	cl, err := nwc.NewClient(c, connectionURL)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}
	// Execute the requested method
	switch method {
	case "get_wallet_service_info":
		handleGetWalletServiceInfo(c, cl)
	case "get_info":
		handleGetInfo(c, cl)
	case "get_balance":
		handleGetBalance(c, cl)
	case "get_budget":
		handleGetBudget(c, cl)
	case "make_invoice":
		handleMakeInvoice(c, cl, args)
	case "pay_invoice":
		handlePayInvoice(c, cl, args)
	case "pay_keysend":
		handlePayKeysend(c, cl, args)
	case "lookup_invoice":
		handleLookupInvoice(c, cl, args)
	case "list_transactions":
		handleListTransactions(c, cl, args)
	case "make_hold_invoice":
		handleMakeHoldInvoice(c, cl, args)
	case "settle_hold_invoice":
		handleSettleHoldInvoice(c, cl, args)
	case "cancel_hold_invoice":
		handleCancelHoldInvoice(c, cl, args)
	case "sign_message":
		handleSignMessage(c, cl, args)
	case "create_connection":
		handleCreateConnection(c, cl, args)
	case "subscribe":
		handleSubscribe(c, cl)
	default:
		fmt.Printf("Unknown method: %s\n", method)
		printUsage()
		os.Exit(1)
	}
}

func handleGetWalletServiceInfo(c context.T, cl *nwc.Client) {
	if _, raw, err := cl.GetWalletServiceInfo(c, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleCancelHoldInvoice(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> cancel_hold_invoice <payment_hash>")
		return
	}

	params := &nwc.CancelHoldInvoiceParams{
		PaymentHash: args[0],
	}
	var err error
	var raw []byte
	if raw, err = cl.CancelHoldInvoice(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleCreateConnection(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 3 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> create_connection <pubkey> <name> <methods> [<notification_types>] [<max_amount>] [<budget_renewal>] [<expires_at>]")
		return
	}
	params := &nwc.CreateConnectionParams{
		Pubkey:         args[0],
		Name:           args[1],
		RequestMethods: strings.Split(args[2], ","),
	}
	if len(args) > 3 {
		params.NotificationTypes = strings.Split(args[3], ",")
	}
	if len(args) > 4 {
		maxAmount, err := strconv.ParseUint(args[4], 10, 64)
		if err != nil {
			fmt.Printf("Error parsing max_amount: %v\n", err)
			return
		}
		params.MaxAmount = &maxAmount
	}
	if len(args) > 5 {
		params.BudgetRenewal = &args[5]
	}
	if len(args) > 6 {
		expiresAt, err := strconv.ParseInt(args[6], 10, 64)
		if err != nil {
			fmt.Printf("Error parsing expires_at: %v\n", err)
			return
		}
		params.ExpiresAt = &expiresAt
	}
	var raw []byte
	var err error
	if raw, err = cl.CreateConnection(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleGetBalance(c context.T, cl *nwc.Client) {
	if _, raw, err := cl.GetBalance(c, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleGetBudget(c context.T, cl *nwc.Client) {
	if _, raw, err := cl.GetBudget(c, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleGetInfo(c context.T, cl *nwc.Client) {
	if _, raw, err := cl.GetInfo(c, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleListTransactions(c context.T, cl *nwc.Client, args []string) {
	params := &nwc.ListTransactionsParams{}
	if len(args) > 0 {
		limit, err := strconv.ParseUint(args[0], 10, 16)
		if err != nil {
			fmt.Printf("Error parsing limit: %v\n", err)
			return
		}
		limitUint16 := uint16(limit)
		params.Limit = &limitUint16
	}
	if len(args) > 1 {
		offset, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			fmt.Printf("Error parsing offset: %v\n", err)
			return
		}
		offsetUint32 := uint32(offset)
		params.Offset = &offsetUint32
	}
	if len(args) > 2 {
		from, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			fmt.Printf("Error parsing from: %v\n", err)
			return
		}
		params.From = &from
	}
	if len(args) > 3 {
		until, err := strconv.ParseInt(args[3], 10, 64)
		if err != nil {
			fmt.Printf("Error parsing until: %v\n", err)
			return
		}
		params.Until = &until
	}
	var raw []byte
	var err error
	if _, raw, err = cl.ListTransactions(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleLookupInvoice(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> lookup_invoice <payment_hash or invoice>")
		return
	}
	params := &nwc.LookupInvoiceParams{}
	// Determine if the argument is a payment hash or an invoice
	if strings.HasPrefix(args[0], "ln") {
		invoice := args[0]
		params.Invoice = &invoice
	} else {
		paymentHash := args[0]
		params.PaymentHash = &paymentHash
	}
	var err error
	var raw []byte
	if _, raw, err = cl.LookupInvoice(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleMakeHoldInvoice(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 2 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> make_hold_invoice <amount> <payment_hash> [<description>] [<description_hash>] [<expiry>]")
		return
	}
	amount, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Printf("Error parsing amount: %v\n", err)
		return
	}
	params := &nwc.MakeHoldInvoiceParams{
		Amount:      amount,
		PaymentHash: args[1],
	}
	if len(args) > 2 {
		params.Description = args[2]
	}
	if len(args) > 3 {
		params.DescriptionHash = args[3]
	}
	if len(args) > 4 {
		expiry, err := strconv.ParseInt(args[4], 10, 64)
		if err != nil {
			fmt.Printf("Error parsing expiry: %v\n", err)
			return
		}
		params.Expiry = &expiry
	}
	var raw []byte
	if _, raw, err = cl.MakeHoldInvoice(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleMakeInvoice(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> make_invoice <amount> [<description>] [<description_hash>] [<expiry>]")
		return
	}
	amount, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Printf("Error parsing amount: %v\n", err)
		return
	}
	params := &nwc.MakeInvoiceParams{
		Amount: amount,
	}
	if len(args) > 1 {
		params.Description = args[1]
	}
	if len(args) > 2 {
		params.DescriptionHash = args[2]
	}
	if len(args) > 3 {
		expiry, err := strconv.ParseInt(args[3], 10, 64)
		if err != nil {
			fmt.Printf("Error parsing expiry: %v\n", err)
			return
		}
		params.Expiry = &expiry
	}
	var raw []byte
	if _, raw, err = cl.MakeInvoice(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handlePayKeysend(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 2 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> pay_keysend <pubkey> <amount> [<preimage>] [<tlv_type> <tlv_value>...]")
		return
	}
	pubkey := args[0]
	amount, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Printf("Error parsing amount: %v\n", err)
		return
	}
	params := &nwc.PayKeysendParams{
		Pubkey: pubkey,
		Amount: amount,
	}
	// Optional preimage
	if len(args) > 2 {
		preimage := args[2]
		params.Preimage = &preimage
	}
	// Optional TLV records (must come in pairs)
	if len(args) > 3 {
		// Start from index 3 and process pairs of arguments
		for i := 3; i < len(args)-1; i += 2 {
			tlvType, err := strconv.ParseUint(args[i], 10, 32)
			if err != nil {
				fmt.Printf("Error parsing TLV type: %v\n", err)
				return
			}
			tlvValue := args[i+1]
			params.TLVRecords = append(
				params.TLVRecords, nwc.PayKeysendTLVRecord{
					Type:  uint32(tlvType),
					Value: tlvValue,
				},
			)
		}
	}
	var raw []byte
	if _, raw, err = cl.PayKeysend(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handlePayInvoice(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> pay_invoice <invoice> [<amount>] [<comment>]")
		return
	}
	params := &nwc.PayInvoiceParams{
		Invoice: args[0],
	}
	if len(args) > 1 {
		amount, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			fmt.Printf("Error parsing amount: %v\n", err)
			return
		}
		params.Amount = &amount
	}
	if len(args) > 2 {
		comment := args[2]
		params.Metadata = &nwc.PayInvoiceMetadata{
			Comment: &comment,
		}
	}
	if _, raw, err := cl.PayInvoice(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleSettleHoldInvoice(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> settle_hold_invoice <preimage>")
		return
	}
	params := &nwc.SettleHoldInvoiceParams{
		Preimage: args[0],
	}
	var raw []byte
	var err error
	if raw, err = cl.SettleHoldInvoice(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleSignMessage(c context.T, cl *nwc.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Missing required arguments")
		fmt.Println("Usage: walletcli <NWC connection URL> sign_message <message>")
		return
	}

	params := &nwc.SignMessageParams{
		Message: args[0],
	}
	var raw []byte
	var err error
	if _, raw, err = cl.SignMessage(c, params, true); !chk.E(err) {
		fmt.Println(string(raw))
	}
}

func handleSubscribe(c context.T, cl *nwc.Client) {
	// Create a context with a cancel
	c, cancel := context.Cancel(c)
	interrupt.AddHandler(cancel)

	// Get wallet service info to check if notifications are supported
	wsi, _, err := cl.GetWalletServiceInfo(c, false)
	if err != nil {
		fmt.Printf("Error getting wallet service info: %v\n", err)
		return
	}

	// Check if the wallet supports notifications
	if len(wsi.NotificationTypes) == 0 {
		fmt.Println("Wallet does not support notifications")
		return
	}
	var evc event.C
	if evc, err = cl.Subscribe(c); chk.E(err) {
		return
	}
	for {
		select {
		case <-c.Done():
			return
		case ev := <-evc:
			fmt.Println(ev.Marshal(nil))
		}
	}
}
