package cLightning

import (
	"github.com/lncm/invoicer/common"
	"github.com/pkg/errors"
)

const ClientName = "clightning"

type CLightning struct{}

func (cLightning CLightning) Invoice(amount float64, desc string) (payment common.Invoice, err error) {
	return payment, errors.New("not implemented yet")
}

func (cLightning CLightning) Status(hash string) (s common.Status, err error) {
	return s, errors.New("not implemented yet")
}

func (cLightning CLightning) Address(bool) (address string, err error) {
	return address, errors.New("not implemented yet")
}

func (cLightning CLightning) Info() (info common.Info, err error) {
	return info, errors.New("not implemented yet")
}

func (cLightning CLightning) History() (invoices common.Invoices, err error) {
	return invoices, errors.New("not implemented yet")
}

func New() CLightning {
	return CLightning{}
}
