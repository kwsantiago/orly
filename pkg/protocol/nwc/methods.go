package nwc

import (
	"bytes"
	"fmt"
	"time"

	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/values"
)

func (cl *Client) GetWalletServiceInfo(c context.T, noUnmarshal bool) (
	wsi *WalletServiceInfo, raw []byte, err error,
) {
	ctx, cancel := context.Timeout(c, 10*time.Second)
	defer cancel()
	var rc *ws.Client
	if rc, err = ws.RelayConnect(c, cl.relay); chk.E(err) {
		return
	}
	var sub *ws.Subscription
	if sub, err = rc.Subscribe(
		ctx, filters.New(
			&filter.F{
				Limit:   values.ToUintPointer(1),
				Kinds:   kinds.New(kind.WalletServiceInfo),
				Authors: tag.New(cl.walletPublicKey),
			},
		),
	); chk.E(err) {
		return
	}
	defer sub.Unsub()
	select {
	case <-ctx.Done():
		err = fmt.Errorf("context canceled")
		return
	case e := <-sub.Events:
		raw = e.Marshal(nil)
		if noUnmarshal {
			return
		}
		wsi = &WalletServiceInfo{}
		encTag := e.Tags.GetFirst(tag.New(EncryptionTag))
		notTag := e.Tags.GetFirst(tag.New(NotificationTag))
		if encTag != nil {
			et := bytes.Split(encTag.Value(), []byte(" "))
			for _, v := range et {
				wsi.EncryptionTypes = append(wsi.EncryptionTypes, v)
			}
		}
		if notTag != nil {
			nt := bytes.Split(notTag.Value(), []byte(" "))
			for _, v := range nt {
				wsi.NotificationTypes = append(wsi.NotificationTypes, v)
			}
		}
		caps := bytes.Split(e.Content, []byte(" "))
		for _, v := range caps {
			wsi.Capabilities = append(wsi.Capabilities, v)
		}
	}
	return
}

func (cl *Client) CancelHoldInvoice(
	c context.T, chi *CancelHoldInvoiceParams, noUnmarshal bool,
) (raw []byte, err error) {
	return cl.RPC(c, CancelHoldInvoice, chi, nil, noUnmarshal, nil)
}

func (cl *Client) CreateConnection(
	c context.T, cc *CreateConnectionParams, noUnmarshal bool,
) (raw []byte, err error) {
	return cl.RPC(c, CreateConnection, cc, nil, noUnmarshal, nil)
}

func (cl *Client) GetBalance(c context.T, noUnmarshal bool) (
	gb *GetBalanceResult, raw []byte, err error,
) {
	gb = &GetBalanceResult{}
	raw, err = cl.RPC(c, GetBalance, nil, gb, noUnmarshal, nil)
	return
}

func (cl *Client) GetBudget(c context.T, noUnmarshal bool) (
	gb *GetBudgetResult, raw []byte, err error,
) {
	gb = &GetBudgetResult{}
	raw, err = cl.RPC(c, GetBudget, nil, gb, noUnmarshal, nil)
	return
}

func (cl *Client) GetInfo(c context.T, noUnmarshal bool) (
	gi *GetInfoResult, raw []byte, err error,
) {
	gi = &GetInfoResult{}
	raw, err = cl.RPC(c, GetInfo, nil, gi, noUnmarshal, nil)
	return
}

func (cl *Client) ListTransactions(
	c context.T, params *ListTransactionsParams, noUnmarshal bool,
) (lt *ListTransactionsResult, raw []byte, err error) {
	lt = &ListTransactionsResult{}
	raw, err = cl.RPC(c, ListTransactions, params, &lt, noUnmarshal, nil)
	return
}

func (cl *Client) LookupInvoice(
	c context.T, params *LookupInvoiceParams, noUnmarshal bool,
) (li *LookupInvoiceResult, raw []byte, err error) {
	li = &LookupInvoiceResult{}
	raw, err = cl.RPC(c, LookupInvoice, params, &li, noUnmarshal, nil)
	return
}

func (cl *Client) MakeHoldInvoice(
	c context.T,
	mhi *MakeHoldInvoiceParams, noUnmarshal bool,
) (mi *MakeInvoiceResult, raw []byte, err error) {
	mi = &MakeInvoiceResult{}
	raw, err = cl.RPC(c, MakeHoldInvoice, mhi, mi, noUnmarshal, nil)
	return
}

func (cl *Client) MakeInvoice(
	c context.T, params *MakeInvoiceParams, noUnmarshal bool,
) (mi *MakeInvoiceResult, raw []byte, err error) {
	mi = &MakeInvoiceResult{}
	raw, err = cl.RPC(c, MakeInvoice, params, &mi, noUnmarshal, nil)
	return
}

// MultiPayInvoice

// MultiPayKeysend

func (cl *Client) PayKeysend(
	c context.T, params *PayKeysendParams, noUnmarshal bool,
) (pk *PayKeysendResult, raw []byte, err error) {
	pk = &PayKeysendResult{}
	raw, err = cl.RPC(c, PayKeysend, params, &pk, noUnmarshal, nil)
	return
}

func (cl *Client) PayInvoice(
	c context.T, params *PayInvoiceParams, noUnmarshal bool,
) (pi *PayInvoiceResult, raw []byte, err error) {
	pi = &PayInvoiceResult{}
	raw, err = cl.RPC(c, PayInvoice, params, &pi, noUnmarshal, nil)
	return
}

func (cl *Client) SettleHoldInvoice(
	c context.T, shi *SettleHoldInvoiceParams, noUnmarshal bool,
) (raw []byte, err error) {
	return cl.RPC(c, SettleHoldInvoice, shi, nil, noUnmarshal, nil)
}

func (cl *Client) SignMessage(
	c context.T, sm *SignMessageParams, noUnmarshal bool,
) (res *SignMessageResult, raw []byte, err error) {
	res = &SignMessageResult{}
	raw, err = cl.RPC(c, SignMessage, sm, &res, noUnmarshal, nil)
	return
}
