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
	InvoiceClient  lnrpc.LightningClient
	ReadOnlyClient lnrpc.LightningClient
}

var (
	lndHost          = flag.String("lnd-host", "localhost", "Specify hostname where your lnd is available")
	lndPort          = flag.Int64("lnd-port", 10009, "Port on which lnd is listening")
	tlsCert          = flag.String("lnd-tls", "tls.cert", "Specify path to tls.cert file")
	invoiceMacaroon  = flag.String("lnd-invoice", "invoice.macaroon", "Specify path to invoice.macaroon file")
	readOnlyMacaroon = flag.String("lnd-readonly", "readonly.macaroon", "Specify path to readonly.macaroon file")
)

func (lnd Lnd) Invoice(amount float64, desc string) (invoice common.Invoice, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inv, err := lnd.InvoiceClient.AddInvoice(ctx, &lnrpc.Invoice{
		Memo:  desc,
		Value: int64(amount * 1e8),
	})
	if err != nil {
		return
	}

	return common.Invoice{
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

	inv, err := lnd.InvoiceClient.LookupInvoice(ctx, &lnrpc.PaymentHash{RHash: invId})
	if err != nil {
		return
	}

	return common.Status{
		Ts:      inv.CreationDate,
		Settled: inv.Settled,
		Expiry:  inv.Expiry,
	}, nil
}

func (lnd Lnd) Info() (info common.Info, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	i, err := lnd.ReadOnlyClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return
	}

	return common.Info{Uris: i.Uris}, nil
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

	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", *lndHost, *lndPort), []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	}...)
	if err != nil {
		panic(errors.Wrapf(err, "unable to connect to %s:%d", *lndHost, *lndPort))
	}

	return lnrpc.NewLightningClient(connection)
}

func New() Lnd {
	// TODO: verify flags(?)
	creds, err := credentials.NewClientTLSFromFile(*tlsCert, *lndHost)
	if err != nil {
		panic(err)
	}

	return Lnd{
		getClient(creds, *invoiceMacaroon),
		getClient(creds, *readOnlyMacaroon),
	}
}
