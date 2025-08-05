package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"orly.dev/pkg/protocol/nwc"
)

func printUsage() {
	fmt.Println("Usage: nwcclient \"<connection URL>\" <method> [parameters...]")
	fmt.Println("\nSupported methods:")
	fmt.Println("  get_info                - Get wallet information")
	fmt.Println("  get_balance             - Get wallet balance")
	fmt.Println("  get_budget              - Get wallet budget")
	fmt.Println("  make_invoice            - Create an invoice (amount, description, [description_hash], [expiry])")
	fmt.Println("  pay_invoice             - Pay an invoice (invoice, [amount])")
	fmt.Println("  pay_keysend             - Send a keysend payment (amount, pubkey, [preimage])")
	fmt.Println("  lookup_invoice          - Look up an invoice (payment_hash or invoice)")
	fmt.Println("  list_transactions       - List transactions ([from], [until], [limit], [offset], [unpaid], [type])")
	fmt.Println("  sign_message            - Sign a message (message)")
	fmt.Println("\nUnsupported methods (due to limitations in the nwc package):")
	fmt.Println("  create_connection       - Create a connection")
	fmt.Println("  make_hold_invoice       - Create a hold invoice")
	fmt.Println("  settle_hold_invoice     - Settle a hold invoice")
	fmt.Println("  cancel_hold_invoice     - Cancel a hold invoice")
	fmt.Println("  multi_pay_invoice       - Pay multiple invoices")
	fmt.Println("  multi_pay_keysend       - Send multiple keysend payments")
	fmt.Println("\nParameters format:")
	fmt.Println("  - Positional parameters are used for required fields")
	fmt.Println("  - For list_transactions, named parameters are used: 'from', 'until', 'limit', 'offset', 'unpaid', 'type'")
	fmt.Println("    Example: nwcclient <url> list_transactions limit 10 type incoming")
	os.Exit(1)
}

func main() {
	// Check if we have enough arguments
	if len(os.Args) < 3 {
		printUsage()
	}

	// Parse connection URL and method
	connectionURL := os.Args[1]
	methodStr := os.Args[2]
	method := nwc.Capability(methodStr)

	// Parse the wallet connect URL
	opts, err := nwc.ParseWalletConnectURL(connectionURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing connection URL: %v\n", err)
		os.Exit(1)
	}

	// Create a new NWC client
	client, err := nwc.NewNWCClient(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating NWC client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Execute the requested method
	var result interface{}

	switch method {
	case nwc.GetInfo:
		result, err = client.GetInfo()

	case nwc.GetBalance:
		result, err = client.GetBalance()

	case nwc.GetBudget:
		result, err = client.GetBudget()

	case nwc.MakeInvoice:
		if len(os.Args) < 5 {
			fmt.Fprintf(
				os.Stderr,
				"Error: make_invoice requires at least amount and description\n",
			)
			printUsage()
		}
		amount, err := strconv.ParseInt(os.Args[3], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing amount: %v\n", err)
			os.Exit(1)
		}
		description := os.Args[4]

		req := &nwc.MakeInvoiceRequest{
			Amount:      amount,
			Description: description,
		}

		// Optional parameters
		if len(os.Args) > 5 {
			req.DescriptionHash = os.Args[5]
		}
		if len(os.Args) > 6 {
			expiry, err := strconv.ParseInt(os.Args[6], 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing expiry: %v\n", err)
				os.Exit(1)
			}
			req.Expiry = &expiry
		}

		result, err = client.MakeInvoice(req)

	case nwc.PayInvoice:
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Error: pay_invoice requires an invoice\n")
			printUsage()
		}

		req := &nwc.PayInvoiceRequest{
			Invoice: os.Args[3],
		}

		// Optional amount parameter
		if len(os.Args) > 4 {
			amount, err := strconv.ParseInt(os.Args[4], 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing amount: %v\n", err)
				os.Exit(1)
			}
			req.Amount = &amount
		}

		result, err = client.PayInvoice(req)

	case nwc.PayKeysend:
		if len(os.Args) < 5 {
			fmt.Fprintf(
				os.Stderr, "Error: pay_keysend requires amount and pubkey\n",
			)
			printUsage()
		}

		amount, err := strconv.ParseInt(os.Args[3], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing amount: %v\n", err)
			os.Exit(1)
		}

		req := &nwc.PayKeysendRequest{
			Amount: amount,
			Pubkey: os.Args[4],
		}

		// Optional preimage
		if len(os.Args) > 5 {
			req.Preimage = os.Args[5]
		}

		result, err = client.PayKeysend(req)

	case nwc.LookupInvoice:
		if len(os.Args) < 4 {
			fmt.Fprintf(
				os.Stderr,
				"Error: lookup_invoice requires a payment_hash or invoice\n",
			)
			printUsage()
		}

		param := os.Args[3]
		req := &nwc.LookupInvoiceRequest{}

		// Determine if the parameter is a payment hash or an invoice
		if strings.HasPrefix(param, "ln") {
			req.Invoice = param
		} else {
			req.PaymentHash = param
		}

		result, err = client.LookupInvoice(req)

	case nwc.ListTransactions:
		req := &nwc.ListTransactionsRequest{}

		// Parse optional parameters
		paramIndex := 3
		for paramIndex < len(os.Args) {
			if paramIndex+1 >= len(os.Args) {
				break
			}

			paramName := os.Args[paramIndex]
			paramValue := os.Args[paramIndex+1]

			switch paramName {
			case "from":
				val, err := strconv.ParseInt(paramValue, 10, 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing from: %v\n", err)
					os.Exit(1)
				}
				req.From = &val
			case "until":
				val, err := strconv.ParseInt(paramValue, 10, 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing until: %v\n", err)
					os.Exit(1)
				}
				req.Until = &val
			case "limit":
				val, err := strconv.ParseInt(paramValue, 10, 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing limit: %v\n", err)
					os.Exit(1)
				}
				req.Limit = &val
			case "offset":
				val, err := strconv.ParseInt(paramValue, 10, 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing offset: %v\n", err)
					os.Exit(1)
				}
				req.Offset = &val
			case "unpaid":
				val := paramValue == "true"
				req.Unpaid = &val
			case "type":
				req.Type = &paramValue
			default:
				fmt.Fprintf(os.Stderr, "Unknown parameter: %s\n", paramName)
				os.Exit(1)
			}

			paramIndex += 2
		}

		result, err = client.ListTransactions(req)

	case nwc.SignMessage:
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Error: sign_message requires a message\n")
			printUsage()
		}

		req := &nwc.SignMessageRequest{
			Message: os.Args[3],
		}

		result, err = client.SignMessage(req)

	case nwc.CreateConnection, nwc.MakeHoldInvoice, nwc.SettleHoldInvoice, nwc.CancelHoldInvoice, nwc.MultiPayInvoice, nwc.MultiPayKeysend:
		fmt.Fprintf(
			os.Stderr,
			"Error: Method %s is not directly supported by the CLI tool.\n",
			methodStr,
		)
		fmt.Fprintf(
			os.Stderr,
			"This is because these methods don't have exported client methods in the nwc package.\n",
		)
		fmt.Fprintf(
			os.Stderr,
			"Only the following methods are currently supported: get_info, get_balance, get_budget, make_invoice, pay_invoice, pay_keysend, lookup_invoice, list_transactions, sign_message\n",
		)
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "Error: Unsupported method: %s\n", methodStr)
		printUsage()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing method: %v\n", err)
		os.Exit(1)
	}

	// Print the result as JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling result to JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}
