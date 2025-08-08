# Mock Wallet Service Examples

This document contains example commands for testing the mock wallet service using the CLI client.

## Starting the Mock Wallet Service

To start the mock wallet service, run the following command from the project root:

```bash
go run cmd/walletcli/mock-wallet-service/main.go --relay ws://localhost:8080 --generate-key
```

This will generate a new wallet key and connect to a relay at ws://localhost:8080. The output will include the wallet's public key, which you'll need for connecting to it.

Alternatively, you can provide your own wallet key:

```bash
go run cmd/walletcli/mock-wallet-service/main.go --relay ws://localhost:8080 --key YOUR_PRIVATE_KEY_HEX
```

## Connecting to the Mock Wallet Service

To connect to the mock wallet service, you'll need to create a connection URL in the following format:

```
nostr+walletconnect://WALLET_PUBLIC_KEY?relay=ws://localhost:8080&secret=CLIENT_SECRET_KEY
```

Where:
- `WALLET_PUBLIC_KEY` is the public key of the wallet service (printed when starting the service)
- `CLIENT_SECRET_KEY` is a private key for the client (you can generate one using any nostr key generation tool)

For example:

```
nostr+walletconnect://7e7e9c42a91bfef19fa929e5fda1b72e0ebc1a4c1141673e2794234d86addf4e?relay=ws://localhost:8080&secret=d5e4f0a6b2c8a9e7d1f3b5a8c2e4f6a8b0d2c4e6f8a0b2d4e6f8a0c2e4d6b8a0
```

## Example Commands

Below are example commands for each method supported by the mock wallet service. Replace `CONNECTION_URL` with your actual connection URL.

### Get Wallet Service Info

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" get_wallet_service_info
```

### Get Info

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" get_info
```

### Get Balance

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" get_balance
```

### Get Budget

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" get_budget
```

### Make Invoice

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" make_invoice 1000 "Test invoice"
```

This creates an invoice for 1000 sats with the description "Test invoice".

### Pay Invoice

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" pay_invoice "lnbc10n1p3zry4app5wkpza973yxheqzh6gr5vt93m3w9mfakz7r35nzk3j6cjgdyvd9ksdqqcqzpgxqyz5vqsp5usyc4lk9chsfp53kvcnvq456ganh60d89reykdngsmtj6yw3nhvq9qyyssqy4lgd8tj274q2rnzl7xvjwh9xct6rkjn47fn7tvj2s8loyy83gy7z5a5xxaqjz3tldmhglggnv8x8h8xwj7gxcr9gy5aquawzh4gqj6d3h4"
```

This pays an invoice. You can use any valid Lightning invoice string.

### Pay Keysend

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" pay_keysend "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" 1000
```

This sends 1000 sats to the specified public key using keysend.

### Lookup Invoice

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" lookup_invoice "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

This looks up an invoice by payment hash.

### List Transactions

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" list_transactions 10
```

This lists up to 10 transactions.

### Make Hold Invoice

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" make_hold_invoice 1000 "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" "Test hold invoice"
```

This creates a hold invoice for 1000 sats with the specified payment hash and description.

### Settle Hold Invoice

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" settle_hold_invoice "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

This settles a hold invoice with the specified preimage.

### Cancel Hold Invoice

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" cancel_hold_invoice "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

This cancels a hold invoice with the specified payment hash.

### Sign Message

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" sign_message "Test message to sign"
```

This signs a message with the wallet's private key.

### Create Connection

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" create_connection "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" "Test Connection" "get_info,get_balance,make_invoice" "payment_received,payment_sent"
```

This creates a connection with the specified public key, name, methods, and notification types.

### Subscribe

```bash
go run cmd/walletcli/main.go "CONNECTION_URL" subscribe
```

This subscribes to notifications from the wallet service.

## Complete Example Workflow

Here's a complete example workflow for testing the mock wallet service:

1. Start the mock wallet service:
   ```bash
   go run cmd/walletcli/mock-wallet-service/main.go --relay ws://localhost:8080 --generate-key
   ```

2. Note the wallet's public key from the output.

3. Generate a client secret key (or use an existing one).

4. Create a connection URL:
   ```
   nostr+walletconnect://WALLET_PUBLIC_KEY?relay=ws://localhost:8080&secret=CLIENT_SECRET_KEY
   ```

5. Get wallet service info:
   ```bash
   go run cmd/walletcli/main.go "CONNECTION_URL" get_wallet_service_info
   ```

6. Get wallet info:
   ```bash
   go run cmd/walletcli/main.go "CONNECTION_URL" get_info
   ```

7. Get wallet balance:
   ```bash
   go run cmd/walletcli/main.go "CONNECTION_URL" get_balance
   ```

8. Create an invoice:
   ```bash
   go run cmd/walletcli/main.go "CONNECTION_URL" make_invoice 1000 "Test invoice"
   ```

9. Look up the invoice:
   ```bash
   go run cmd/walletcli/main.go "CONNECTION_URL" lookup_invoice "PAYMENT_HASH_FROM_INVOICE"
   ```

10. Subscribe to notifications:
    ```bash
    go run cmd/walletcli/main.go "CONNECTION_URL" subscribe
    ```

## Notes

- The mock wallet service returns generic results for all methods, regardless of the input parameters.
- The mock wallet service does not actually perform any real Lightning Network operations.
- The mock wallet service does not persist any data between restarts.