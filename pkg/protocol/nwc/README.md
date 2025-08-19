# NWC Client

Nostr Wallet Connect (NIP-47) client implementation.

## Usage

```go
import "orly.dev/pkg/protocol/nwc"

// Create client from NWC connection URI
client, err := nwc.NewClient("nostr+walletconnect://...")
if err != nil {
    log.Fatal(err)
}

// Make requests
var info map[string]any
err = client.Request(ctx, "get_info", nil, &info)

var balance map[string]any
err = client.Request(ctx, "get_balance", nil, &balance)

var invoice map[string]any
params := map[string]any{"amount": 1000, "description": "test"}
err = client.Request(ctx, "make_invoice", params, &invoice)
```

## Methods

- `get_info` - Get wallet info
- `get_balance` - Get wallet balance  
- `make_invoice` - Create invoice
- `lookup_invoice` - Check invoice status
- `pay_invoice` - Pay invoice

## Payment Notifications

```go
// Subscribe to payment notifications
err = client.SubscribeNotifications(ctx, func(notificationType string, notification map[string]any) error {
    if notificationType == "payment_received" {
        amount := notification["amount"].(float64)
        description := notification["description"].(string)
        // Process payment...
    }
    return nil
})
```

## Features

- NIP-44 encryption
- Event signing
- Relay communication
- Payment notifications
- Error handling