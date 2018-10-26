package clightning

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lncm/invoicer/common"
	"github.com/pkg/errors"
	"os/exec"
)

const ClientName = "clightning"

type Clightning struct {
	Binary  string
	Network string
}

func clightningInfo(c *gin.Context) {
	info, err := exec.Command("/usr/bin/docker", "exec", "lightningpay", "lightning-cli", "getinfo").Output()
	if err == nil {
		c.String(200, fmt.Sprintf("%s", info))
		return
	}

	c.JSON(500, gin.H{
		"error": fmt.Sprintf("Error from lightning service: %s", err),
	})
	return

}

func (clightning Clightning) Invoice(amount float64, desc string) (invoice common.Invoice, err error) {
	return invoice, errors.New("not implemented yet")
}

func (clightning Clightning) Status(hash string) (s common.Status, err error) {
	return s, errors.New("not implemented yet")
}

func (clightning Clightning) Info() (info common.Info, err error) {
	return info, errors.New("not implemented yet")
}

func New(binary, network string) Clightning {
	return Clightning{binary, network}
}
