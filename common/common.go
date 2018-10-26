package common

import "time"

const (
	DefaultUsersFile     = "./users.list"
	MaxDescLen           = 639
	DefaultInvoiceExpiry = 180
)

type (
	Invoice struct {
		Hash   string `json:"r_hash"`
		Bolt11 string `json:"pay_req"`
	}

	Status struct {
		Ts      int64 `json:"creation_date,string"`
		Settled bool  `json:"settled"`
		Expiry  int64 `json:"expiry,string"`
	}

	Info struct {
		Uris []string `json:"uris"`
	}
)

func (s Status) IsExpired() bool {
	return time.Now().After(time.Unix(s.Ts+s.Expiry, 0))
}
