package nwc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"orly.dev/pkg/crypto/encryption"
	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
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
	"time"
)

type Client struct {
	pool            *ws.Pool
	relays          []string
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
	cl = &Client{
		pool:            ws.NewPool(c),
		relays:          parts.relays,
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
	c context.T, method Capability, params, result any, opts *rpcOptions,
) (err error) {
	timeout := time.Duration(10)
	if opts == nil && opts.timeout == nil {
		timeout = *opts.timeout
	}
	ctx, cancel := context.Timeout(c, timeout)
	defer cancel()
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
			tag.New([]byte("p"), cl.walletPublicKey),
			tag.New("encryption", "nip44_v2"),
		),
	}
	if err = ev.Sign(cl.clientSecretKey); chk.E(err) {
		return
	}
	hasWorked := make(chan struct{})
	evs := cl.pool.SubMany(
		c, cl.relays, &filters.T{
			F: []*filter.F{
				{
					Limit:   values.ToUintPointer(1),
					Kinds:   kinds.New(kind.WalletRequest),
					Authors: tag.New(cl.walletPublicKey),
					Tags:    tags.New(tag.New([]byte("#e"), ev.ID)),
				},
			},
		},
	)
	for _, u := range cl.relays {
		go func(u string) {
			var relay *ws.Client
			if relay, err = cl.pool.EnsureRelay(u); chk.E(err) {
				return
			}
			if err = relay.Publish(c, ev); chk.E(err) {
				return
			}
			select {
			case hasWorked <- struct{}{}:
			case <-ctx.Done():
				err = fmt.Errorf("context canceled waiting for request send")
				return
			default:
			}
		}(u)
	}
	select {
	case <-hasWorked:
	// continue
	case <-ctx.Done():
		err = fmt.Errorf("timed out waiting for relays")
		return
	}
	select {
	case <-ctx.Done():
		err = fmt.Errorf("context canceled waiting for response")
	case e := <-evs:
		var plain []byte
		if plain, err = encryption.Decrypt(
			e.Event.Content, cl.conversationKey,
		); chk.E(err) {
			return
		}
		resp := &Response{
			Result: &result,
		}
		if err = json.Unmarshal(plain, resp); chk.E(err) {
			return
		}
		return
	}
	return
}

func (cl *Client) GetWalletServiceInfo(c context.T) (
	wsi *WalletServiceInfo, err error,
) {
	lim := uint(1)
	evc := cl.pool.SubMany(
		c, cl.relays, &filters.T{
			F: []*filter.F{
				{
					Limit:   &lim,
					Kinds:   kinds.New(kind.WalletInfo),
					Authors: tag.New(cl.walletPublicKey),
				},
			},
		},
	)
	select {
	case <-c.Done():
		err = fmt.Errorf("GetWalletServiceInfo canceled")
		return
	case ev := <-evc:
		var encryptionTypes []EncryptionType
		var notificationTypes []NotificationType
		encryptionTag := ev.Event.Tags.GetFirst(tag.New("encryption"))
		notificationsTag := ev.Event.Tags.GetFirst(tag.New("notifications"))
		if encryptionTag != nil {
			et := encryptionTag.ToSliceOfBytes()
			encType := bytes.Split(et[0], []byte(" "))
			for _, e := range encType {
				encryptionTypes = append(encryptionTypes, e)
			}
		}
		if notificationsTag != nil {
			nt := notificationsTag.ToSliceOfBytes()
			notifs := bytes.Split(nt[0], []byte(" "))
			for _, e := range notifs {
				notificationTypes = append(notificationTypes, e)
			}
		}
		cp := bytes.Split(ev.Event.Content, []byte(" "))
		var capabilities []Capability
		for _, capability := range cp {
			capabilities = append(capabilities, capability)
		}
		wsi = &WalletServiceInfo{
			EncryptionTypes:   encryptionTypes,
			NotificationTypes: notificationTypes,
			Capabilities:      capabilities,
		}
	}
	return
}

func (cl *Client) GetInfo(c context.T) (gi *GetInfoResult, err error) {
	gi = &GetInfoResult{}
	if err = cl.RPC(c, GetInfo, nil, gi, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) MakeInvoice(
	c context.T, params *MakeInvoiceParams,
) (mi *MakeInvoiceResult, err error) {
	mi = &MakeInvoiceResult{}
	if err = cl.RPC(c, MakeInvoice, params, &mi, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) PayInvoice(
	c context.T, params *PayInvoiceParams,
) (pi *PayInvoiceResult, err error) {
	pi = &PayInvoiceResult{}
	if err = cl.RPC(c, PayInvoice, params, &pi, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) LookupInvoice(
	c context.T, params *LookupInvoiceParams,
) (li *LookupInvoiceResult, err error) {
	li = &LookupInvoiceResult{}
	if err = cl.RPC(c, LookupInvoice, params, &li, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) ListTransactions(
	c context.T, params *ListTransactionsParams,
) (lt *ListTransactionsResult, err error) {
	lt = &ListTransactionsResult{}
	if err = cl.RPC(c, ListTransactions, params, &lt, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) GetBalance(c context.T) (gb *GetBalanceResult, err error) {
	gb = &GetBalanceResult{}
	if err = cl.RPC(c, GetBalance, nil, gb, nil); chk.E(err) {
		return
	}
	return
}
