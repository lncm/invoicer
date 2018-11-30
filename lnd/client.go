package lnd

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/lncm/invoicer/common"
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
// TODO: lnd-specific flags
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
	// TODO: make sure this one uses `readonly.macaroon` instead
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	i, err := lnd.ReadOnlyClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return
	}
	fmt.Printf("%#v\n", i)

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

	connection, err := grpc.Dial("reckless.nolim1t.co:10009", []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	}...)
	if err != nil {
		panic(err)
	}

	return lnrpc.NewLightningClient(connection)
}

func New() Lnd {
	creds, err := credentials.NewClientTLSFromFile("tls.cert", "reckless.nolim1t.co")
	if err != nil {
		panic(err)
	}
	return Lnd{
		getClient(creds, "invoice.macaroon"),
		getClient(creds, "readonly.macaroon"),
	}
}
