package cLightning

import (
	"context"

	"github.com/lncm/invoicer/common"
	"golang.org/x/xerrors"
)

const ClientName = "clightning"

type CLightning struct{}

func (cLightning CLightning) NewInvoice(ctx context.Context, amount int64, desc string) (invoice, hash string, err error) {
	return invoice, hash, xerrors.New("not implemented yet")
}

func (cLightning CLightning) Status(ctx context.Context, hash string) (s common.Status, err error) {
	return s, xerrors.New("not implemented yet")
}

func (cLightning CLightning) StatusWait(ctx context.Context, hash string) (s common.Status, err error) {
	return s, xerrors.New("not implemented yet")
}

func (cLightning CLightning) NewAddress(context.Context, bool) (address string, err error) {
	return address, xerrors.New("not implemented yet")
}

func (cLightning CLightning) Info(ctx context.Context) (info common.Info, err error) {
	return info, xerrors.New("not implemented yet")
}

func (cLightning CLightning) History(ctx context.Context) (invoices common.Invoices, err error) {
	return invoices, xerrors.New("not implemented yet")
}

func New() CLightning {
	return CLightning{}
}
