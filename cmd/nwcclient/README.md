# NWC Client CLI Tool

A command-line interface tool for making calls to Nostr Wallet Connect (NWC) services.

## Overview

This CLI tool allows you to interact with NWC wallet services using the methods defined in the NIP-47 specification. It provides a simple interface for executing wallet operations and displays the JSON response from the wallet service.

## Usage

```
nwcclient <connection URL> <method> [parameters...]
```

### Connection URL

The connection URL should be in the Nostr Wallet Connect format:

```
nostr+walletconnect://<wallet_pubkey>?relay=<relay_url>&secret=<secret>
```

### Supported Methods

The following methods are supported by this CLI tool:

- `get_info` - Get wallet information
- `get_balance` - Get wallet balance
- `get_budget` - Get wallet budget
- `make_invoice` - Create an invoice
- `pay_invoice` - Pay an invoice
- `pay_keysend` - Send a keysend payment
- `lookup_invoice` - Look up an invoice
- `list_transactions` - List transactions
- `sign_message` - Sign a message

### Unsupported Methods

The following methods are defined in the NIP-47 specification but are not directly supported by this CLI tool due to limitations in the underlying nwc package:

- `create_connection` - Create a connection
- `make_hold_invoice` - Create a hold invoice
- `settle_hold_invoice` - Settle a hold invoice
- `cancel_hold_invoice` - Cancel a hold invoice
- `multi_pay_invoice` - Pay multiple invoices
- `multi_pay_keysend` - Send multiple keysend payments

## Method Parameters

### Methods with No Parameters

- `get_info`
- `get_balance`
- `get_budget`

Example:
```
nwcclient <connection URL> get_info
```

### Methods with Parameters

#### make_invoice

```
nwcclient <connection URL> make_invoice <amount> <description> [description_hash] [expiry]
```

- `amount` - Amount in millisatoshis (msats)
- `description` - Invoice description
- `description_hash` (optional) - Hash of the description
- `expiry` (optional) - Expiry time in seconds

Example:
```
nwcclient <connection URL> make_invoice 1000000 "Test invoice" "" 3600
```

#### pay_invoice

```
nwcclient <connection URL> pay_invoice <invoice> [amount]
```

- `invoice` - BOLT11 invoice
- `amount` (optional) - Amount in millisatoshis (msats)

Example:
```
nwcclient <connection URL> pay_invoice lnbc1...
```

#### pay_keysend

```
nwcclient <connection URL> pay_keysend <amount> <pubkey> [preimage]
```

- `amount` - Amount in millisatoshis (msats)
- `pubkey` - Recipient's public key
- `preimage` (optional) - Payment preimage

Example:
```
nwcclient <connection URL> pay_keysend 1000000 03...
```

#### lookup_invoice

```
nwcclient <connection URL> lookup_invoice <payment_hash_or_invoice>
```

- `payment_hash_or_invoice` - Payment hash or BOLT11 invoice

Example:
```
nwcclient <connection URL> lookup_invoice 3d...
```

#### list_transactions

```
nwcclient <connection URL> list_transactions [from <timestamp>] [until <timestamp>] [limit <count>] [offset <count>] [unpaid <true|false>] [type <incoming|outgoing>]
```

Parameters are specified as name-value pairs:

- `from` - Start timestamp
- `until` - End timestamp
- `limit` - Maximum number of transactions to return
- `offset` - Number of transactions to skip
- `unpaid` - Whether to include unpaid transactions
- `type` - Transaction type (incoming or outgoing)

Example:
```
nwcclient <connection URL> list_transactions limit 10 type incoming
```

#### sign_message

```
nwcclient <connection URL> sign_message <message>
```

- `message` - Message to sign

Example:
```
nwcclient <connection URL> sign_message "Hello, world!"
```

## Output

The tool prints the JSON response from the wallet service to stdout. If an error occurs, an error message is printed to stderr.

## Limitations

- The tool only supports methods that have direct client methods in the nwc package.
- Complex parameters like metadata are not supported.
- The tool does not support interactive authentication or authorization.