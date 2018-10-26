package lnd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/lncm/invoicer/common"
	"github.com/pkg/errors"
	"os/exec"
)

const ClientName = "lnd"

type Lnd struct {
	Binary  string
	Network string
}

// TODO: change all `.Run()` to `.Output()` below

func (lnd Lnd) Invoice(amount float64, desc string) (invoice common.Invoice, err error) {
	cmd := exec.Command(
		lnd.Binary,
		fmt.Sprintf("--network=%s", lnd.Network),
		"addinvoice",
		fmt.Sprintf("--expiry=%d", common.DefaultInvoiceExpiry), // TODO: allow for custom expiry on invoices
		fmt.Sprintf("--memo=%s", desc),                          // TODO: sanitize `desc` better
		fmt.Sprintf("%d", int(amount)),
	)

	var out, err2 bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err2

	err = cmd.Run()
	if err != nil {
		return invoice, errors.Wrap(err, err2.String())
	}

	err = json.NewDecoder(&out).Decode(&invoice)
	if err != nil {
		return invoice, errors.Wrap(err, "unable to decode response")
	}

	return
}

func (lnd Lnd) Status(hash string) (s common.Status, err error) {
	cmd := exec.Command(
		lnd.Binary,
		fmt.Sprintf("--network=%s", lnd.Network),
		"lookupinvoice",
		hash,
	)

	var out, err2 bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err2

	err = cmd.Run()
	if err != nil {
		return s, errors.Wrap(err, err2.String())
	}

	err = json.NewDecoder(&out).Decode(&s)
	if err != nil {
		return s, errors.Wrap(err, "unable to decode response")
	}

	return s, nil
}

func (lnd Lnd) Info() (info common.Info, err error) {
	cmd := exec.Command(
		lnd.Binary,
		fmt.Sprintf("--network=%s", lnd.Network),
		"getinfo",
	)

	var out, err2 bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err2

	err = cmd.Run()
	if err != nil {
		return info, errors.Wrap(err, err2.String())
	}

	err = json.NewDecoder(&out).Decode(&info)
	if err != nil {
		return info, errors.Wrap(err, "unable to decode response")
	}

	return info, nil
}

func New(binary, network string) Lnd {
	return Lnd{binary, network}
}
