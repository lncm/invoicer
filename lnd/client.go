package lnd

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/lncm/invoicer/common"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"io/ioutil"
	"time"
)

const ClientName = "lnd"

type Lnd struct {
	invoiceClient  lnrpc.LightningClient
	readOnlyClient lnrpc.LightningClient
}

var (
	hostname         = flag.String("lnd-host", "localhost", "Specify hostname where your lnd is available")
	port             = flag.Int64("lnd-port", 10009, "Port on which lnd is listening")
	tlsCert          = flag.String("lnd-tls", "tls.cert", "Specify path to tls.cert file")
	invoiceMacaroon  = flag.String("lnd-invoice", "invoice.macaroon", "Specify path to invoice.macaroon file")
	readOnlyMacaroon = flag.String("lnd-readonly", "readonly.macaroon", "Specify path to readonly.macaroon file")
)

func (lnd Lnd) Invoice(amount float64, desc string) (_ common.LnInvoice, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inv, err := lnd.invoiceClient.AddInvoice(ctx, &lnrpc.Invoice{
		Memo:   desc,
		Value:  int64(amount),
		Expiry: common.DefaultInvoiceExpiry,
	})
	if err != nil {
		return
	}

	return common.LnInvoice{
		Hash:   hex.EncodeToString(inv.RHash),
		Bolt11: inv.PaymentRequest,
	}, nil
}

func (lnd Lnd) Status(hash string) (s common.Status, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	invId, err := hex.DecodeString(hash)
	if err != nil {
		return
	}

	inv, err := lnd.invoiceClient.LookupInvoice(ctx, &lnrpc.PaymentHash{RHash: invId})
	if err != nil {
		return
	}

	return common.Status{
		Ts:      inv.CreationDate,
		Settled: inv.Settled,
		Expiry:  inv.Expiry,
		Value:   inv.Value,
	}, nil
}

func (lnd Lnd) Address(bech32 bool) (address string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrType := lnrpc.NewAddressRequest_NESTED_PUBKEY_HASH
	if bech32 {
		addrType = lnrpc.NewAddressRequest_WITNESS_PUBKEY_HASH
	}

	addrResp, err := lnd.invoiceClient.NewAddress(ctx, &lnrpc.NewAddressRequest{
		Type: addrType,
	})
	if err != nil {
		return
	}

	return addrResp.Address, nil
}

func (lnd Lnd) Info() (info common.Info, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	i, err := lnd.readOnlyClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return
	}

	return common.Info{Uris: i.Uris}, nil
}

func (lnd Lnd) History() (invoices common.Invoices, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := lnd.readOnlyClient.ListInvoices(ctx, &lnrpc.ListInvoiceRequest{
		NumMaxInvoices: 250,
	})
	if err != nil {
		return
	}

	for _, inv := range list.Invoices {
		invoices = append(invoices, common.Invoice{
			Bolt11:      inv.PaymentRequest,
			Description: inv.Memo,
			Hash:        hex.EncodeToString(inv.RHash),
			Amount:      inv.AmtPaidSat,
			Paid:        inv.Settled,
			PaidAt:      inv.SettleDate,
			Expired:     inv.Expiry < time.Now().Unix(),
			ExpireAt:    inv.CreationDate + inv.Expiry,
		})
	}

	// TODO: optionally reverse order(?)

	return
}

func getClient(creds credentials.TransportCredentials, file string) lnrpc.LightningClient {
	macaroonBytes, err := ioutil.ReadFile(file)
	if err != nil {
		panic(fmt.Sprintln("Cannot read macaroon file", err))
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macaroonBytes); err != nil {
		panic(fmt.Sprintln("Cannot unmarshal macaroon", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", *hostname, *port), []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	}...)
	if err != nil {
		panic(errors.Wrapf(err, "unable to connect to %s:%d", *hostname, *port))
	}

	return lnrpc.NewLightningClient(connection)
}

func New() Lnd {
	creds, err := credentials.NewClientTLSFromFile(*tlsCert, *hostname)
	if err != nil {
		panic(err)
	}

	return Lnd{
		getClient(creds, *invoiceMacaroon),
		getClient(creds, *readOnlyMacaroon),
	}
}
