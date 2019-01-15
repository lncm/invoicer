package common

import "time"

const (
	DefaultUsersFile     = "./users.list"
	DefaultInvoiceExpiry = 180
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

	Status struct {
		Ts      int64
		Settled bool
		Expiry  int64
		Value   int64
	}

	Invoice struct {
		Bolt11      string `json:"bolt11"`
		Description string `json:"description"`
		Hash        string `json:"hash"`
		Amount      int64  `json:"amount,omitempty"`
		Paid        bool   `json:"is_paid"`
		PaidAt      int64  `json:"paid_at,omitempty"`
		Expired     bool   `json:"is_expired"`
		ExpireAt    int64  `json:"expire_at,omitempty"`
	}

	Invoices []Invoice

	Info struct {
		Uris []string `json:"uris"`
	}

	AddrStatus struct {
		Address       string   `json:"address"`
		Amount        float64  `json:"amount"`
		Confirmations int64    `json:"confirmations"`
		Label         string   `json:"label"`
		TxIds         []string `json:"txids"`
	}

	AddrsStatus []AddrStatus
)

func (s Status) IsExpired() bool {
	return time.Now().After(time.Unix(s.Ts+s.Expiry, 0))
}
