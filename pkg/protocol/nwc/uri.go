package nwc

import (
	"errors"
	"net/url"

	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/utils/chk"
)

type ConnectionParams struct {
	clientSecretKey []byte
	walletPublicKey []byte
	relay           string
}

// GetWalletPublicKey returns the wallet public key from the ConnectionParams.
func (c *ConnectionParams) GetWalletPublicKey() []byte {
	return c.walletPublicKey
}

func ParseConnectionURI(nwcUri string) (parts *ConnectionParams, err error) {
	var p *url.URL
	if p, err = url.Parse(nwcUri); chk.E(err) {
		return
	}
	parts = &ConnectionParams{}
	if p.Scheme != "nostr+walletconnect" {
		err = errors.New("incorrect scheme")
		return
	}
	if parts.walletPublicKey, err = p256k.HexToBin(p.Host); chk.E(err) {
		err = errors.New("invalid public key")
		return
	}
	query := p.Query()
	var ok bool
	var relay []string
	if relay, ok = query["relay"]; !ok {
		err = errors.New("missing relay parameter")
		return
	}
	if len(relay) == 0 {
		return nil, errors.New("no relays")
	}
	parts.relay = relay[0]
	var secret string
	if secret = query.Get("secret"); secret == "" {
		err = errors.New("missing secret parameter")
		return
	}
	if parts.clientSecretKey, err = p256k.HexToBin(secret); chk.E(err) {
		err = errors.New("invalid secret")
		return
	}
	return
}
