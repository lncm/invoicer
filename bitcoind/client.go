package bitcoind

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/xerrors"

	"github.com/lncm/invoicer/common"
)

const (
	DefaultHostname = "localhost"
	DefaultPort     = 8332
	DefaultUsername = "invoicer"

	MethodGetBlockCount        = "getblockcount"
	MethodGetNewAddress        = "getnewaddress"
	MethodImportAddress        = "importaddress"
	MethodListReceiveByAddress = "listreceivedbyaddress"

	Bech32 = "bech32"
)

type (
	Bitcoind struct {
		url, user, pass string
	}

	requestBody struct {
		JsonRpc string        `json:"jsonrpc"`
		Id      string        `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	responseBody struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code,omitempty"`
			Message string `json:"message,omitempty"`
		} `json:"error,omitempty"`
	}
)

func (b Bitcoind) BlockCount() (count int64, err error) {
	res, err := b.sendRequest(MethodGetBlockCount)
	err = json.Unmarshal(res, &count)
	return
}

func (b Bitcoind) Address(bech32 bool) (addr string, err error) {
	var params []interface{}
	if bech32 {
		params = []interface{}{"", Bech32}
	}

	res, err := b.sendRequest(MethodGetNewAddress, params...)
	if err != nil {
		return
	}

	err = json.Unmarshal(res, &addr)
	return
}

func (b Bitcoind) ImportAddress(address, label string) (err error) {
	_, err = b.sendRequest(MethodImportAddress, address, label, false)
	return
}

// NOTE: returns all if empty string passed
func (b Bitcoind) CheckAddress(address string) (state common.AddrsStatus, err error) {
	params := []interface{}{0, true, true}
	if address != "" {
		params = append(params, address)
	}

	res, err := b.sendRequest(MethodListReceiveByAddress, params...)
	if err != nil {
		return
	}

	err = json.Unmarshal(res, &state)
	return
}

func (b Bitcoind) sendRequest(method string, params ...interface{}) (response []byte, err error) {
	reqBody, err := json.Marshal(requestBody{
		JsonRpc: "1.0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", b.url, bytes.NewReader(reqBody))
	if err != nil {
		return
	}

	req.SetBasicAuth(b.user, b.pass)
	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	defer func() { _ = res.Body.Close() }()

	resBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	var resBody responseBody
	err = json.Unmarshal(resBytes, &resBody)
	if err != nil {
		return
	}

	if resBody.Error != nil {
		return nil, xerrors.Errorf("bitcoind error (%d): %w", resBody.Error.Code, resBody.Error.Message)
	}

	return resBody.Result, nil
}

func New(conf common.Bitcoind) (Bitcoind, error) {
	if conf.Host == "" {
		conf.Host = DefaultHostname
	}

	if conf.Port == 0 {
		conf.Port = DefaultPort
	}

	if conf.User == "" {
		conf.User = DefaultUsername
	}

	client := Bitcoind{
		url:  fmt.Sprintf("http://%s:%d", conf.Host, conf.Port),
		user: conf.User,
		pass: conf.Pass,
	}

	_, err := client.BlockCount()
	if err != nil {
		return Bitcoind{}, xerrors.Errorf("can't connect to Bitcoind: %w", err)
	}

	return client, nil
}
