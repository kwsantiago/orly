package nwc

import (
	"encoding/json"
	"fmt"
	"time"

	"orly.dev/pkg/crypto/encryption"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/values"
)

type Client struct {
	relay           string
	clientSecretKey signer.I
	walletPublicKey []byte
	conversationKey []byte
}

func NewClient(connectionURI string) (cl *Client, err error) {
	var parts *ConnectionParams
	if parts, err = ParseConnectionURI(connectionURI); chk.E(err) {
		return
	}
	cl = &Client{
		relay:           parts.relay,
		clientSecretKey: parts.clientSecretKey,
		walletPublicKey: parts.walletPublicKey,
		conversationKey: parts.conversationKey,
	}
	return
}

func (cl *Client) Request(c context.T, method string, params, result any) (err error) {
	ctx, cancel := context.Timeout(c, 10*time.Second)
	defer cancel()

	request := map[string]any{"method": method}
	if params != nil {
		request["params"] = params
	}

	var req []byte
	if req, err = json.Marshal(request); chk.E(err) {
		return
	}

	var content []byte
	if content, err = encryption.Encrypt(req, cl.conversationKey); chk.E(err) {
		return
	}

	ev := &event.E{
		Content:   content,
		CreatedAt: timestamp.Now(),
		Kind:      kind.New(23194),
		Tags: tags.New(
			tag.New("encryption", "nip44_v2"),
			tag.New("p", hex.Enc(cl.walletPublicKey)),
		),
	}

	if err = ev.Sign(cl.clientSecretKey); chk.E(err) {
		return
	}

	var rc *ws.Client
	if rc, err = ws.RelayConnect(ctx, cl.relay); chk.E(err) {
		return
	}
	defer rc.Close()

	var sub *ws.Subscription
	if sub, err = rc.Subscribe(
		ctx, filters.New(
			&filter.F{
				Limit: values.ToUintPointer(1),
				Kinds: kinds.New(kind.New(23195)),
				Since: &timestamp.T{V: time.Now().Unix()},
			},
		),
	); chk.E(err) {
		return
	}
	defer sub.Unsub()

	if err = rc.Publish(ctx, ev); chk.E(err) {
		return fmt.Errorf("publish failed: %w", err)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("no response from wallet (connection may be inactive)")
	case e := <-sub.Events:
		if e == nil {
			return fmt.Errorf("subscription closed (wallet connection inactive)")
		}
		if len(e.Content) == 0 {
			return fmt.Errorf("empty response content")
		}
		var raw []byte
		if raw, err = encryption.Decrypt(e.Content, cl.conversationKey); chk.E(err) {
			return fmt.Errorf("decryption failed (invalid conversation key): %w", err)
		}

		var resp map[string]any
		if err = json.Unmarshal(raw, &resp); chk.E(err) {
			return
		}

		if errData, ok := resp["error"].(map[string]any); ok {
			code, _ := errData["code"].(string)
			msg, _ := errData["message"].(string)
			return fmt.Errorf("%s: %s", code, msg)
		}

		if result != nil && resp["result"] != nil {
			var resultBytes []byte
			if resultBytes, err = json.Marshal(resp["result"]); chk.E(err) {
				return
			}
			if err = json.Unmarshal(resultBytes, result); chk.E(err) {
				return
			}
		}
	}

	return
}