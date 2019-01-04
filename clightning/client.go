package cLightning

import (
	"github.com/lncm/invoicer/common"
	"github.com/pkg/errors"
)

const ClientName = "clightning"

type CLightning struct{}

func (cLightning CLightning) Invoice(amount float64, desc string) (invoice common.Invoice, err error) {
	return invoice, errors.New("not implemented yet")
}

func (cLightning CLightning) Status(hash string) (s common.Status, err error) {
	return s, errors.New("not implemented yet")
}

func (cLightning CLightning) Info() (info common.Info, err error) {
	return info, errors.New("not implemented yet")
}

func New() CLightning {
	return CLightning{}
}
