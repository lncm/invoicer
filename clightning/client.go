package clightning

import (
	"context"
	"errors"

	"github.com/lncm/invoicer/common"
)

const ClientName = "clightning"

type CLightning struct{}

func (cLightning CLightning) NewInvoice(ctx context.Context, amount int64, desc string) (invoice, hash string, err error) {
	return invoice, hash, errors.New("not implemented yet")
}

func (cLightning CLightning) Status(ctx context.Context, hash string) (s common.Status, err error) {
	return s, errors.New("not implemented yet")
}

func (cLightning CLightning) StatusWait(ctx context.Context, hash string) (s common.Status, err error) {
	return s, errors.New("not implemented yet")
}

func (cLightning CLightning) NewAddress(context.Context, bool) (address string, err error) {
	return address, errors.New("not implemented yet")
}

func (cLightning CLightning) Info(ctx context.Context) (info common.Info, err error) {
	return info, errors.New("not implemented yet")
}

func (cLightning CLightning) History(ctx context.Context) (invoices common.Invoices, err error) {
	return invoices, errors.New("not implemented yet")
}

func New() CLightning {
	return CLightning{}
}
