package lnd

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"

	"github.com/lncm/invoicer/common"
)

const (
	ClientName = "lnd"

	DefaultHostname  = "localhost"
	DefaultPort      = 10009
	DefaultTls       = "~/.invoicer/tls.cert"
	DefaultInvoice   = "~/.invoicer/invoice.macaroon"
	DefaultReadOnly  = "~/.invoicer/readonly.macaroon"
	DefaultKillCount = 4
)

type Lnd struct {
	// Used to generate invoices and monitor their status
	invoiceClient lnrpc.LightningClient

	// Used to access history and lnd's connection string
	readOnlyClient lnrpc.LightningClient

	notifier InvoiceMonitor
}

func (lnd Lnd) NewInvoice(ctx context.Context, amount int64, desc string) (invoice, hash string, err error) {
	inv, err := lnd.invoiceClient.AddInvoice(ctx, &lnrpc.Invoice{
		Memo:   desc,
		Value:  int64(amount),
		Expiry: common.DefaultInvoiceExpiry,
	})
	if err != nil {
		return
	}

	return inv.PaymentRequest, hex.EncodeToString(inv.RHash), nil
}

func (lnd Lnd) StatusWait(ctx context.Context, hash string) (s common.Status, err error) {
	inv, err := lnd.notifier.Status(ctx, hash)
	if err != nil {
		return common.Status{}, err
	}

	val := inv.Value
	if val == 0 {
		val = inv.AmtPaidSat
	}

	return common.Status{
		Ts:      inv.CreationDate,
		Settled: inv.State == lnrpc.Invoice_SETTLED,
		Expiry:  inv.Expiry,
		Value:   val,
	}, nil
}

func (lnd Lnd) Status(ctx context.Context, hash string) (s common.Status, err error) {
	invId, err := hex.DecodeString(hash)
	if err != nil {
		return
	}

	inv, err := lnd.invoiceClient.LookupInvoice(ctx, &lnrpc.PaymentHash{RHash: invId})
	if err != nil {
		return
	}

	val := inv.Value
	if val == 0 {
		val = inv.AmtPaidSat
	}

	return common.Status{
		Ts:      inv.CreationDate,
		Settled: inv.State == lnrpc.Invoice_SETTLED,
		Expiry:  inv.Expiry,
		Value:   inv.Value,
	}, nil
}

func (lnd Lnd) NewAddress(ctx context.Context, bech32 bool) (address string, err error) {
	addrType := lnrpc.AddressType_NESTED_PUBKEY_HASH
	if bech32 {
		addrType = lnrpc.AddressType_WITNESS_PUBKEY_HASH
	}

	addrResp, err := lnd.invoiceClient.NewAddress(ctx, &lnrpc.NewAddressRequest{
		Type: addrType,
	})
	if err != nil {
		return
	}

	return addrResp.Address, nil
}

func (lnd Lnd) Info(ctx context.Context) (info common.Info, err error) {
	i, err := lnd.readOnlyClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return
	}

	return common.Info{Uris: i.Uris}, nil
}

func (lnd Lnd) History(ctx context.Context) (invoices common.Invoices, err error) {
	// group, c := errgroup.WithContext(ctx)
	//
	// var invoiceList *lnrpc.ListInvoiceResponse
	// group.Go(func() (err error) {
	invoiceList, err := lnd.readOnlyClient.ListInvoices(ctx, &lnrpc.ListInvoiceRequest{
		NumMaxInvoices: 100,
		Reversed:       true,
	})

	// 	return
	// })
	//
	// var chainList *lnrpc.TransactionDetails
	// group.Go(func() (err error) {
	// 	chainList, err = lnd.readOnlyClient.GetTransactions(c, &lnrpc.GetTransactionsRequest{})
	// 	return
	// })
	//
	// err = group.Wait()
	if err != nil {
		return
	}

	for _, inv := range invoiceList.Invoices {
		invoices = append(invoices, common.Invoice{
			Description: inv.Memo,
			Amount:      inv.Value,
			Paid:        inv.State == lnrpc.Invoice_SETTLED,
			PaidAt:      inv.SettleDate,
			Expired:     inv.CreationDate+inv.Expiry < time.Now().Unix(),
			NewPayment: common.NewPayment{
				Bolt11:    inv.PaymentRequest,
				Hash:      hex.EncodeToString(inv.RHash),
				CreatedAt: inv.CreationDate,
				Expiry:    inv.Expiry,
			},
		})
	}

	return
}

func (lnd Lnd) checkConnectionStatus(killCount int) {
	if killCount == 0 {
		return
	}

	failures := 0

	for {
		failures++

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := lnd.Info(ctx)
		if err == nil {
			failures = 0
		}

		cancel()

		if failures > 3 {
			log.WithField("count", failures).WithField("final", true).Panic("lnd unreachable")
		}

		if failures > 0 {
			log.WithField("count", failures).Printf("lnd unreachable")
		}

		time.Sleep(time.Minute)
	}
}

func getClient(creds credentials.TransportCredentials, fullHostname, file string) lnrpc.LightningClient {
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

	connection, err := grpc.DialContext(ctx, fullHostname, []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	}...)
	if err != nil {
		panic(errors.Wrapf(err, "unable to connect to %s", fullHostname))
	}

	return lnrpc.NewLightningClient(connection)
}

func New(conf common.Lnd) (lnd Lnd) {
	if conf.Host == "" {
		conf.Host = DefaultHostname
	}

	if conf.Port == 0 {
		conf.Port = DefaultPort
	}

	if conf.Tls == "" {
		conf.Tls = DefaultTls
	}
	conf.Tls = common.CleanAndExpandPath(conf.Tls)

	if conf.Macaroons.Invoice == "" {
		conf.Macaroons.Invoice = DefaultInvoice
	}
	conf.Macaroons.Invoice = common.CleanAndExpandPath(conf.Macaroons.Invoice)

	if conf.Macaroons.ReadOnly == "" {
		conf.Macaroons.ReadOnly = DefaultReadOnly
	}
	conf.Macaroons.ReadOnly = common.CleanAndExpandPath(conf.Macaroons.ReadOnly)

	creds, err := credentials.NewClientTLSFromFile(conf.Tls, conf.Host)
	if err != nil {
		panic(err)
	}

	hostname := fmt.Sprintf("%s:%d", conf.Host, conf.Port)

	invoiceClient := getClient(creds, hostname, conf.Macaroons.Invoice)
	notifier, err := NewNotifier(invoiceClient)
	if err != nil {
		panic(err)
	}

	lnd = Lnd{
		invoiceClient:  invoiceClient,
		readOnlyClient: getClient(creds, hostname, conf.Macaroons.ReadOnly),
		notifier:       notifier,
	}

	if conf.KillCount == nil {
		// `hah` assignment is silly, but necessary…
		hah := DefaultKillCount
		conf.KillCount = &hah
	}

	go lnd.checkConnectionStatus(*conf.KillCount)

	return
}
