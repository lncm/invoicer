package common

import "time"

const (
	DefaultUsersFile     = "./users.list"
	MaxDescLen           = 639
	DefaultInvoiceExpiry = 180
)

type (
	NewPayment struct {
		Bolt11  string `json:"bolt11"`
		Hash    string `json:"hash"`
		Address string `json:"address,omitempty"`
	}

	Status struct {
		Ts      int64
		Settled bool
		Expiry  int64
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
)

func (s Status) IsExpired() bool {
	return time.Now().After(time.Unix(s.Ts+s.Expiry, 0))
}
