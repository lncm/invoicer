package ln

import (
	"context"
	"encoding/hex"
	"errors"

	log "github.com/sirupsen/logrus"

	lnrpc "github.com/lncm/invoicer/ln/lnd"
)

type (
	subscriber struct {
		hash    string
		invoice chan *lnrpc.Invoice
	}

	InvoiceMonitor struct {
		lnClient lnrpc.LightningClient
		subs     chan []subscriber
	}
)

func (im InvoiceMonitor) checkForInvoices(invSub lnrpc.Lightning_SubscribeInvoicesClient) {
	for {
		var inv *lnrpc.Invoice
		inv, err := invSub.Recv()
		if err != nil {
			log.WithError(err).Error("invoice subscriber service has failed")
			return
		}

		im.notifyAll(inv)
	}
}

func (im InvoiceMonitor) start() error {
	ctx := context.Background()

	invSub, err := im.lnClient.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{})
	if err != nil {
		return err
	}

	im.subs <- []subscriber{}

	go im.checkForInvoices(invSub)

	return nil
}

func (im InvoiceMonitor) add(hash string, status chan *lnrpc.Invoice) {
	im.subs <- append(<-im.subs, subscriber{
		hash:    hash,
		invoice: status,
	})
}

func (im InvoiceMonitor) remove(hash string, status chan *lnrpc.Invoice) {
	subs := <-im.subs

	var remainingSubs []subscriber
	for _, sub := range subs {
		if sub.invoice != status {
			remainingSubs = append(remainingSubs, sub)
		}
	}

	im.subs <- remainingSubs
}

func (im InvoiceMonitor) notifyAll(inv *lnrpc.Invoice) {
	s := <-im.subs

	var remainingSubs []subscriber
	for _, x := range s {
		if x.hash == hex.EncodeToString(inv.RHash) {
			x.invoice <- inv
			close(x.invoice)
			continue
		}

		remainingSubs = append(remainingSubs, x)
	}

	im.subs <- remainingSubs
}

func (im InvoiceMonitor) Status(ctx context.Context, hash string) (*lnrpc.Invoice, error) {
	status := make(chan *lnrpc.Invoice)

	im.add(hash, status)

	select {
	case s := <-status:
		return s, nil

	case <-ctx.Done():
		im.remove(hash, status)
		return nil, ctx.Err()
	}
}

func NewNotifier(client lnrpc.LightningClient) (InvoiceMonitor, error) {
	if client == nil {
		return InvoiceMonitor{}, errors.New("valid Lightning Client has to be provided")
	}

	n := InvoiceMonitor{
		lnClient: client,
		subs:     make(chan []subscriber, 1),
	}

	err := n.start()
	if err != nil {
		return InvoiceMonitor{}, err
	}

	return n, nil
}
