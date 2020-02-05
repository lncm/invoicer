package ln

import (
	"testing"

	"github.com/lncm/invoicer/common"
)

func TestNew(t *testing.T) {
	conf := common.LndConfig{
		Host:      "localhost",
		Port:      10009,
		TLS:       "../tls.cert",
		Macaroons: common.Macaroons{},
	}

	_, _ = Start(conf)
}
