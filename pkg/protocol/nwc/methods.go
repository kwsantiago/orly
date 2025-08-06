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
	timeout := 10 * time.Second
	ctx, cancel := context.Timeout(c, timeout)
	defer cancel()
	var rc *ws.Client
	if rc, err = ws.RelayConnect(c, cl.relay); chk.E(err) {
		return
	}
	if err = rc.Connect(c); chk.E(err) {
		return
	}
	var sub *ws.Subscription
	if sub, err = rc.Subscribe(
		ctx, filters.New(
			&filter.F{
				Limit:   values.ToUintPointer(1),
				Kinds:   kinds.New(kind.WalletRequest),
				Authors: tag.New(cl.walletPublicKey),
			},
		),
	); chk.E(err) {
		return
	}
	defer sub.Unsub()
	select {
	case <-c.Done():
		err = fmt.Errorf("GetWalletServiceInfo canceled")
		return
	case ev := <-sub.Events:
		var encryptionTypes []EncryptionType
		var notificationTypes []NotificationType
		encryptionTag := ev.Tags.GetFirst(tag.New("encryption"))
		notificationsTag := ev.Tags.GetFirst(tag.New("notifications"))
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
		cp := bytes.Split(ev.Content, []byte(" "))
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
