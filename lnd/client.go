package lnd

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
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
	DefaultTLS       = "~/.invoicer/tls.cert"
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
		Value:  amount,
		Expiry: common.DefaultInvoiceExpiry,
	})
	if err != nil {
		return
	}

	return inv.GetPaymentRequest(), hex.EncodeToString(inv.GetRHash()), nil
}

func (lnd Lnd) StatusWait(ctx context.Context, hash string) (s common.Status, err error) {
	inv, err := lnd.notifier.Status(ctx, hash)
	if err != nil {
		return common.Status{}, err
	}

	val := inv.GetValue()
	if val == 0 {
		val = inv.GetAmtPaidSat()
	}

	return common.Status{
		Ts:      inv.GetCreationDate(),
		Settled: inv.GetState() == lnrpc.Invoice_SETTLED,
		Expiry:  inv.GetExpiry(),
		Value:   val,
	}, nil
}

func (lnd Lnd) Status(ctx context.Context, hash string) (s common.Status, err error) {
	invID, err := hex.DecodeString(hash)
	if err != nil {
		return
	}

	inv, err := lnd.invoiceClient.LookupInvoice(ctx, &lnrpc.PaymentHash{RHash: invID})
	if err != nil {
		return
	}

	val := inv.GetValue()
	if val == 0 {
		val = inv.GetAmtPaidSat()
	}

	return common.Status{
		Ts:      inv.GetCreationDate(),
		Settled: inv.GetState() == lnrpc.Invoice_SETTLED,
		Expiry:  inv.GetExpiry(),
		Value:   val,
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

	return addrResp.GetAddress(), nil
}

func (lnd Lnd) Info(ctx context.Context) (info common.Info, err error) {
	i, err := lnd.readOnlyClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return
	}

	return common.Info{Uris: i.GetUris()}, nil
}

func (lnd Lnd) History(ctx context.Context) (invoices common.Invoices, err error) {
	invoiceList, err := lnd.readOnlyClient.ListInvoices(ctx, &lnrpc.ListInvoiceRequest{
		NumMaxInvoices: 100,
		Reversed:       true,
	})
	if err != nil {
		return
	}

	for _, inv := range invoiceList.Invoices {
		invoices = append(invoices, common.Invoice{
			Description: inv.GetMemo(),
			Amount:      inv.GetValue(),
			Paid:        inv.GetState() == lnrpc.Invoice_SETTLED,
			PaidAt:      inv.GetSettleDate(),
			Expired:     inv.GetCreationDate()+inv.GetExpiry() < time.Now().Unix(),
			NewPayment: common.NewPayment{
				Bolt11:    inv.GetPaymentRequest(),
				Hash:      hex.EncodeToString(inv.GetRHash()),
				CreatedAt: inv.GetCreationDate(),
				Expiry:    inv.GetExpiry(),
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
			if failures > 1 {
				log.WithField("count", failures).Printf("lnd connection reestablished")
			}

			failures = 0
		}

		cancel()

		if failures >= killCount {
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
		panic(fmt.Errorf("unable to connect to %s: %w", fullHostname, err))
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

	if conf.TLS == "" {
		conf.TLS = DefaultTLS
	}
	conf.TLS = common.CleanAndExpandPath(conf.TLS)

	if conf.Macaroons.Invoice == "" {
		conf.Macaroons.Invoice = DefaultInvoice
	}
	conf.Macaroons.Invoice = common.CleanAndExpandPath(conf.Macaroons.Invoice)

	if conf.Macaroons.ReadOnly == "" {
		conf.Macaroons.ReadOnly = DefaultReadOnly
	}
	conf.Macaroons.ReadOnly = common.CleanAndExpandPath(conf.Macaroons.ReadOnly)

	creds, err := credentials.NewClientTLSFromFile(conf.TLS, conf.Host)
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

	return lnd
}
