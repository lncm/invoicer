package common

import (
	"time"

	"github.com/pelletier/go-toml"
)

const (
	DefaultConfigDir  = "~/.lncm/"
	DefaultConfigFile = DefaultConfigDir + "invoicer.conf"
	DefaultLogFile    = DefaultConfigDir + "invoicer.log"

	DefaultInvoiceExpiry = 3600
	MaxInvoiceDescLen    = 639
)

type (
	NewPayment struct {
		CreatedAt int64  `json:"created_at"`
		Expiry    int64  `json:"expiry"`
		Bolt11    string `json:"bolt11"`
		Hash      string `json:"hash"`
		Address   string `json:"address"`
	}

	Payment struct {
		NewPayment

		Description string `json:"description"`

		// The requested amount for the payment
		Amount int64 `json:"amount"`

		// General status of the payment
		Expired bool  `json:"is_expired"`
		Paid    bool  `json:"is_paid"`
		PaidAt  int64 `json:"paid_at,omitempty"`

		// LN specific
		LnPaid bool `json:"ln_paid"`

		// BTC specific
		BtcPaid       bool     `json:"btc_paid"` // only true if amount >= the requested one
		BtcAmount     int64    `json:"btc_amount"`
		Confirmations int64    `json:"confirmations"`
		TxIds         []string `json:"txids"`
	}

	Invoice struct {
		NewPayment

		Description string `json:"description"`

		// What was the requested amount for the payment
		Amount int64 `json:"amount"`

		// general status of the payment
		Expired bool  `json:"is_expired"`
		Paid    bool  `json:"is_paid"`
		PaidAt  int64 `json:"paid_at"`
	}

	Invoices []Invoice

	Status struct {
		Ts      int64 `json:"created_at"`
		Settled bool  `json:"is_paid"`
		Expiry  int64 `json:"expiry"`
		Value   int64 `json:"amount"`
	}

	Info struct {
		OnChain  bool     `json:"on-chain"`
		OffChain bool     `json:"off-chain"`
		Uris     []string `json:"uris"`
	}

	AddrStatus struct {
		Address       string   `json:"address"`
		Amount        float64  `json:"amount"`
		Confirmations int64    `json:"confirmations"`
		Label         string   `json:"label,omitempty"`
		TxIds         []string `json:"txids"`
	}

	AddrsStatus []AddrStatus

	StatusReply struct {
		Code    int         `json:"-"`
		Error   string      `json:"error,omitempty"`
		Ln      *Status     `json:"ln,omitempty"`
		Bitcoin *AddrStatus `json:"bitcoin,omitempty"`
	}
)

func (s Status) IsExpired() bool {
	return time.Now().After(time.Unix(s.Ts+s.Expiry, 0))
}

func (p *Payment) ApplyLn(invoice Invoice) {
	p.NewPayment = invoice.NewPayment
	p.Description = invoice.Description
	p.Amount = invoice.Amount
	p.Expired = invoice.Expired
	p.Expiry = invoice.Expiry
	p.LnPaid = invoice.Paid

	p.Paid = p.Paid || invoice.Paid

	p.checkBtcPaid()
}

func (p *Payment) ApplyBtc(s AddrStatus) {
	p.Address = s.Address
	p.BtcAmount = int64(s.Amount * 1e8)
	p.Confirmations = s.Confirmations
	p.TxIds = s.TxIds

	p.checkBtcPaid()
}

// can only be done after amount is known
func (p *Payment) checkBtcPaid() {
	if p.Amount == 0 || p.BtcAmount == 0 {
		return
	}

	if p.BtcAmount >= p.Amount {
		p.BtcPaid = true
		p.Paid = true
	}
}

// deprecatedConfigLocationCheck checks for a config file in a previously default location.  Checking there will be
//      removed in one of the next versions.
//
//      tl;dr: `mkdir ~/.lncm  &&  mv ~/.invoicer/* ~/.lncm/  &&  rmdir ~/.invoicer`
//
// OR, if you run in Docker, just change where your volume is mounted, ex:
//      `-v $(pwd)/:/root/.invoicer/`    ->    `-v $(pwd)/:/root/.lncm/`
func DeprecatedConfigLocationCheck(path string, err error) (*toml.Tree, error) {
	if path != DefaultConfigFile {
		return nil, err
	}

	const deprecatedLocation = "~/.invoicer/invoicer.conf"

	tree, err2 := toml.LoadFile(CleanAndExpandPath(deprecatedLocation))
	if err2 != nil {
		return nil, err
	}

	return tree, nil
}
