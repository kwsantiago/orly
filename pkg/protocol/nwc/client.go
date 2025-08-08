package nwc

import (
	"encoding/json"
	"fmt"
	"time"

	"orly.dev/pkg/crypto/encryption"
	"orly.dev/pkg/crypto/p256k"
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
	client          *ws.Client
	relay           string
	clientSecretKey signer.I
	walletPublicKey []byte
	conversationKey []byte // nip44
}

type Request struct {
	Method string `json:"method"`
	Params any    `json:"params"`
}

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (err *ResponseError) Error() string {
	return fmt.Sprintf("%s %s", err.Code, err.Message)
}

type Response struct {
	ResultType string         `json:"result_type"`
	Error      *ResponseError `json:"error"`
	Result     any            `json:"result"`
}

func NewClient(c context.T, connectionURI string) (cl *Client, err error) {
	var parts *ConnectionParams
	if parts, err = ParseConnectionURI(connectionURI); chk.E(err) {
		return
	}
	clientKey := &p256k.Signer{}
	if err = clientKey.InitSec(parts.clientSecretKey); chk.E(err) {
		return
	}
	var ck []byte
	if ck, err = encryption.GenerateConversationKeyWithSigner(
		clientKey,
		parts.walletPublicKey,
	); chk.E(err) {
		return
	}
	var relay *ws.Client
	if relay, err = ws.RelayConnect(c, parts.relay); chk.E(err) {
		return
	}
	cl = &Client{
		client:          relay,
		relay:           parts.relay,
		clientSecretKey: clientKey,
		walletPublicKey: parts.walletPublicKey,
		conversationKey: ck,
	}
	return
}

type rpcOptions struct {
	timeout *time.Duration
}

func (cl *Client) RPC(
	c context.T, method Capability, params, result any, noUnmarshal bool,
	opts *rpcOptions,
) (raw []byte, err error) {
	var req []byte
	if req, err = json.Marshal(
		Request{
			Method: string(method),
			Params: params,
		},
	); chk.E(err) {
		return
	}
	var content []byte
	if content, err = encryption.Encrypt(req, cl.conversationKey); chk.E(err) {
		return
	}
	ev := &event.E{
		Content:   content,
		CreatedAt: timestamp.Now(),
		Kind:      kind.WalletRequest,
		Tags: tags.New(
			tag.New("p", hex.Enc(cl.walletPublicKey)),
			tag.New(EncryptionTag, Nip44V2),
		),
	}
	if err = ev.Sign(cl.clientSecretKey); chk.E(err) {
		return
	}
	var rc *ws.Client
	if rc, err = ws.RelayConnect(c, cl.relay); chk.E(err) {
		return
	}
	defer rc.Close()
	var sub *ws.Subscription
	if sub, err = rc.Subscribe(
		c, filters.New(
			&filter.F{
				Limit:   values.ToUintPointer(1),
				Kinds:   kinds.New(kind.WalletResponse),
				Authors: tag.New(cl.walletPublicKey),
				Tags:    tags.New(tag.New("#e", hex.Enc(ev.ID))),
			},
		),
	); chk.E(err) {
		return
	}
	defer sub.Unsub()
	if err = rc.Publish(context.Bg(), ev); chk.E(err) {
		return
	}
	select {
	case <-c.Done():
		err = fmt.Errorf("context canceled waiting for response")
	case e := <-sub.Events:
		if raw, err = encryption.Decrypt(
			e.Content, cl.conversationKey,
		); chk.E(err) {
			return
		}
		if noUnmarshal {
			return
		}
		resp := &Response{
			Result: &result,
		}
		if err = json.Unmarshal(raw, resp); chk.E(err) {
			return
		}
	}
	return
}

func (cl *Client) Subscribe(c context.T) (evc event.C, err error) {
	var rc *ws.Client
	if rc, err = ws.RelayConnect(c, cl.relay); chk.E(err) {
		return
	}
	defer rc.Close()
	var sub *ws.Subscription
	if sub, err = rc.Subscribe(
		c, filters.New(
			&filter.F{
				Kinds: kinds.New(
					kind.WalletNotification, kind.WalletNotificationNip4,
				),
				Authors: tag.New(cl.walletPublicKey),
			},
		),
	); chk.E(err) {
		return
	}
	defer sub.Unsub()
	go func() {
		for {
			select {
			case <-c.Done():
				return
			case ev := <-sub.Events:
				evc <- ev
			}
		}
	}()
	return
}
