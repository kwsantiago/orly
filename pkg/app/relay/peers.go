package relay

import (
	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/encoders/bech32encoding"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/keys"
	"orly.dev/pkg/utils/log"
	"strings"
)

// Peers is a structure that keeps the information required when peer
// replication is enabled.
//
// - Addresses are the relay addresses that will be pushed new events when
// accepted. From ORLY_PEER_RELAYS first field after the |.
//
// - Pubkeys are the relay peer public keys that we will send any event to
// including privileged type. From ORLY_PEER_RELAYS before the |.
//
// - I - the signer of this relay, generated from the nsec in
// ORLY_SECRET_KEY.
type Peers struct {
	Addresses []string
	Pubkeys   [][]byte
	signer.I
}

// Init accepts the lists which will come from config.C for peer relay settings
// and populate the Peers with this data after decoding it.
func (p *Peers) Init(
	addresses []string, sec string,
) (err error) {
	for _, address := range addresses {
		if len(address) == 0 {
			continue
		}
		split := strings.Split(address, "@")
		if len(split) != 2 {
			log.E.F("invalid peer address: %s", address)
			continue
		}
		p.Addresses = append(p.Addresses, split[1])
		var pk []byte
		if pk, err = keys.DecodeNpubOrHex(split[0]); chk.D(err) {
			continue
		}
		p.Pubkeys = append(p.Pubkeys, pk)
		log.I.F("peer %s added; pubkey: %0x", split[1], pk)
	}
	p.I = &p256k.Signer{}
	var s []byte
	if s, err = keys.DecodeNsecOrHex(sec); chk.E(err) {
		return
	}
	if err = p.I.InitSec(s); chk.E(err) {
		return
	}
	var npub []byte
	if npub, err = bech32encoding.BinToNpub(p.I.Pub()); chk.E(err) {
		return
	}
	log.I.F(
		"relay peer initialized, relay's npub: %s",
		npub,
	)
	return
}
