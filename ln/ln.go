package ln

import (
	"context"

	"github.com/lncm/invoicer/common"
)

type LightningClient interface {
	NewAddress(ctx context.Context, bech32 bool) (string, error)
	Info(ctx context.Context) (common.Info, error)
	NewInvoice(ctx context.Context, amount int64, desc string) (string, string, error)
	Status(ctx context.Context, hash string) (common.Status, error)
	StatusWait(ctx context.Context, hash string) (common.Status, error)
	History(ctx context.Context) (common.Invoices, error)
}

func Start(conf common.LndConfig) (Lnd, error) {
	return startClient(conf)
}
