package nwc

import (
	"bytes"
	"fmt"

	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
)

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

func (cl *Client) CancelHoldInvoice(
	c context.T, chi *CancelHoldInvoiceParams,
) (err error) {
	if err = cl.RPC(c, CancelHoldInvoice, chi, nil, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) CreateConnection(
	c context.T, cc *CreateConnectionParams,
) (err error) {
	if err = cl.RPC(c, CreateConnection, cc, nil, nil); chk.E(err) {
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

func (cl *Client) GetBudget(c context.T) (gb *GetBudgetResult, err error) {
	if err = cl.RPC(c, CreateConnection, nil, gb, nil); chk.E(err) {
		return
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

func (cl *Client) ListTransactions(
	c context.T, params *ListTransactionsParams,
) (lt *ListTransactionsResult, err error) {
	lt = &ListTransactionsResult{}
	if err = cl.RPC(c, ListTransactions, params, &lt, nil); chk.E(err) {
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

func (cl *Client) MakeHoldInvoice(
	c context.T,
	mhi *MakeHoldInvoiceParams,
) (mi *MakeInvoiceResult, err error) {
	mi = &MakeInvoiceResult{}
	if err = cl.RPC(c, GetBalance, mhi, mi, nil); chk.E(err) {
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

// MultiPayInvoice

// MultiPayKeysend

func (cl *Client) PayInvoice(
	c context.T, params *PayInvoiceParams,
) (pi *PayInvoiceResult, err error) {
	pi = &PayInvoiceResult{}
	if err = cl.RPC(c, PayInvoice, params, &pi, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) SettleHoldInvoice(
	c context.T, shi *SettleHoldInvoiceParams,
) (err error) {
	if err = cl.RPC(c, SettleHoldInvoice, shi, nil, nil); chk.E(err) {
		return
	}
	return
}

func (cl *Client) SignMessage(
	c context.T, sm *SignMessageParams,
) (res *SignMessageResult, err error) {
	res = &SignMessageResult{}
	if err = cl.RPC(c, SignMessage, sm, &res, nil); chk.E(err) {
		return
	}
	return
}
