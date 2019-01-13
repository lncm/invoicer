package bitcoind

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/lncm/invoicer/common"
	"io/ioutil"
	"net/http"
)

var (
	hostname = flag.String("bitcoind-host", "localhost", "Specify hostname where your bitcoind is available")
	port     = flag.Int64("bitcoind-port", 8332, "Port on which bitcoind is listening")
	user     = flag.String("bitcoind-user", "invoicer", "RPC user for bitcoind")
	pass     = flag.String("bitcoind-pass", "", "RPC password for bitcoind")
)

type (
	Bitcoind struct {
		url, user, pass string
	}

	Body struct {
		JsonRpc string        `json:"jsonrpc"`
		Id      string        `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"` // TODO: ???
	}

	rawResponse struct {
		Result json.RawMessage `json:"result"`
		Error  struct {
			Code    int    `json:"code,omitempty"`
			Message string `json:"message,omitempty"`
		} `json:"error,omitempty"`
	}
)

func (b Bitcoind) BlockCount() (count int64, err error) {
	result, err := b.sendRequest("getblockcount")
	err = json.Unmarshal(result, &count)
	return
}

func (b Bitcoind) Address(bech32 bool) (addr string, err error) {
	var params []interface{}
	if bech32 {
		params = []interface{}{"", "bech32"}
	}

	result, err := b.sendRequest("getnewaddress", params...)
	if err != nil {
		return
	}

	err = json.Unmarshal(result, &addr)
	return
}

func (b Bitcoind) ImportAddress(address, label string) (err error) {
	_, err = b.sendRequest("importaddress", address, label, false)
	return
}

func (b Bitcoind) CheckAddress(address string) (state common.AddressStatus, err error) {
	params := []interface{}{0, true, true}
	if address != "" {
		params = append(params, address)
	}

	result, err := b.sendRequest("listreceivedbyaddress", params...)
	if err != nil {
		return
	}

	err = json.Unmarshal(result, &state)
	return
}

// TODO: cleanup
func (b Bitcoind) sendRequest(method string, params ...interface{}) (response []byte, err error) {
	body, err := json.Marshal(Body{
		JsonRpc: "1.0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return
	}

	httpReq, err := http.NewRequest("POST", b.url, bytes.NewReader(body))
	if err != nil {
		return
	}

	httpReq.Close = true
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(b.user, b.pass)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return
	}

	// TODO: handle(?)
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var resp2 rawResponse
	err = json.Unmarshal(respBytes, &resp2)
	if err != nil {
		return
	}

	return resp2.Result, nil
}

func New() Bitcoind {
	return Bitcoind{
		url:  fmt.Sprintf("http://%s:%d", *hostname, *port),
		user: *user,
		pass: *pass,
	}
}
