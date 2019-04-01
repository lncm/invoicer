package main

import (
	"context"
	"flag"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/lncm/invoicer/bitcoind"
	"github.com/lncm/invoicer/clightning"
	"github.com/lncm/invoicer/common"
	"github.com/lncm/invoicer/lnd"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
)

type (
	BitcoinClient interface {
		Address(bech32 bool) (string, error)
		BlockCount() (int64, error)
		ImportAddress(address, label string) error
		CheckAddress(address string) (common.AddrsStatus, error)
	}

	LightningClient interface {
		Address(ctx context.Context, bech32 bool) (string, error)
		Info(ctx context.Context) (common.Info, error)
		NewInvoice(ctx context.Context, amount int64, desc string) (string, string, error)
		Status(ctx context.Context, hash string) (common.Status, error)
		StatusWait(ctx context.Context, hash string) (common.Status, error)
		History(ctx context.Context) (common.Invoices, error)
	}
)

const DefaultInvoicerPort = 8080

var (
	version,
	gitHash string

	lnClient  LightningClient
	btcClient BitcoinClient
	conf      common.Config

	configFilePath = flag.String("config", common.DefaultConfigFile, "Path to a config file in TOML format")

	accounts gin.Accounts
)

func init() {
	flag.Parse()

	// Expand configFile file path and load it
	configFile, err := toml.LoadFile(common.CleanAndExpandPath(*configFilePath))
	if err != nil {
		panic(fmt.Sprintf("unable to load %s:\n\t%v", *configFilePath, err))
	}

	err = configFile.Unmarshal(&conf)
	if err != nil {
		panic(fmt.Sprintf("unable to process %s:\n\t%v", *configFilePath, err))
	}

	// Use lnd when no client is specified
	if conf.LnClient == "" {
		conf.LnClient = lnd.ClientName
	}

	// init specified LN client
	switch strings.ToLower(conf.LnClient) {
	case lnd.ClientName:
		lnClient = lnd.New(conf.Lnd)

	case cLightning.ClientName:
		//lnClient = cLightning.New()

	default:
		panic(fmt.Sprintf("invalid ln-client specified: %s", conf.LnClient))
	}

	// init  BTC client for monitoring on-chain payments
	btcClient = bitcoind.New(conf.Bitcoind)

	versionString := "debug"
	if version != "" && gitHash != "" {
		versionString = fmt.Sprintf("%s (git: %s)", version, gitHash)
	}

	fmt.Printf("version: %s\nLN client: %s\n\n", versionString, conf.LnClient)

	if len(conf.Users) > 0 {
		accounts = gin.Accounts(conf.Users)
	}
}

func newPayment(c *gin.Context) {
	var data struct {
		Amount      int64  `json:"amount"`
		Description string `json:"desc"`
		Only        string `json:"only"`
	}

	err := c.ShouldBindJSON(&data)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	if data.Only != "" && data.Only != "btc" && data.Only != "ln" {
		c.AbortWithStatusJSON(400, gin.H{
			"error": "only= is an optional parameter that can only take `btc` and `ln` as values",
		})
		return
	}

	var payment common.NewPayment

	if data.Only != "btc" {
		if len(data.Description) > common.MaxInvoiceDescLen {
			c.AbortWithStatusJSON(400, gin.H{
				"error": fmt.Sprintf("description too long. Max length is %d.", common.MaxInvoiceDescLen),
			})
			return
		}

		// Generate new LN invoice
		payment.Bolt11, payment.Hash, err = lnClient.NewInvoice(c, data.Amount, data.Description)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"error": errors.WithMessage(err, "can't create new LN invoice").Error(),
			})
			return
		}

		// Extract invoice's creation date & expiry
		invoice, err := lnClient.Status(c, payment.Hash)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"error": errors.WithMessage(err, "can't get LN invoice").Error(),
			})
			return
		}
		payment.CreatedAt = invoice.Ts
		payment.Expiry = invoice.Expiry
	}

	if data.Only != "ln" {
		// get BTC address
		payment.Address, err = lnClient.Address(c, false)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"error": errors.WithMessage(err, "can't get Bitcoin address").Error(),
			})
			return
		}

		label := data.Description
		if len(payment.Hash) > 0 {
			label = payment.Hash
		}

		err = btcClient.ImportAddress(payment.Address, label)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"error": fmt.Sprintf("can't import address (%s) to Bitcoin node: %v", payment.Address, err),
			})
			return
		}
	}

	c.JSON(200, payment)
}

func checkLn(ctx context.Context, hash string) *common.StatusReply {
	lnStatus, err := lnClient.StatusWait(ctx, hash)
	if err != nil {
		return &common.StatusReply{
			Code:  500,
			Error: fmt.Sprintf("unable to fetch invoice: %s", err),
		}
	}

	if lnStatus.Settled {
		return &common.StatusReply{
			Code: 200, Ln: &lnStatus,
		}
	}

	if lnStatus.IsExpired() {
		return &common.StatusReply{
			Code:  408,
			Error: "expired",
		}
	}

	return nil
}

func checkBtc(fin time.Time, addr string, lnProvided bool, desiredAmount int64) *common.StatusReply {
	for time.Now().Before(fin) {
		time.Sleep(2 * time.Second)

		btcStatuses, err := btcClient.CheckAddress(addr)
		if err != nil {
			if !lnProvided {
				return &common.StatusReply{
					Code:  500,
					Error: fmt.Sprintf("unable to check status: %s", err),
				}
			}

			// if LN hash available and fetching bitcoin status produced an error, disable checking bitcoin
			return nil
		}

		btcStatus := btcStatuses[0]

		receivedAmount := int64(btcStatus.Amount) * 1e8
		if btcStatus.Amount == 0 {
			continue
		}

		// no need to return it now; might be useful later
		btcStatus.Label = ""

		if desiredAmount == receivedAmount {
			return &common.StatusReply{
				Code:    200,
				Bitcoin: &btcStatus,
			}
		}

		if receivedAmount > desiredAmount {
			return &common.StatusReply{
				Code:    202,
				Bitcoin: &btcStatus,
			}

		}

		if desiredAmount > receivedAmount {
			return &common.StatusReply{
				Code:    402,
				Error:   "not enough",
				Bitcoin: &btcStatus,
			}
		}
	}

	return nil
}

func status(c *gin.Context) {
	hash := c.Query("hash")
	addr := c.Query("address")

	if len(hash) == 0 && len(addr) == 0 {
		c.AbortWithStatusJSON(500, common.StatusReply{
			Error: "At least one of `hash` or `address` needs to be provided",
		})
		return
	}

	var desiredAmount int64

	// do initial LN invoice check, and adjust expiration if available
	fin := time.Now().Add(common.DefaultInvoiceExpiry * time.Second)
	if len(hash) > 0 {
		lnStatus, err := lnClient.Status(c, hash)
		if err != nil {
			c.AbortWithStatusJSON(500, common.StatusReply{
				Error: fmt.Sprintf("unable to fetch invoice: %s", err),
			})
			return
		}

		if lnStatus.Settled {
			c.JSON(200, common.StatusReply{Ln: &lnStatus})
			return
		}

		if lnStatus.IsExpired() {
			c.AbortWithStatusJSON(408, common.StatusReply{Error: "expired"})
			return
		}

		fin = time.Unix(lnStatus.Ts, 0).Add(time.Duration(lnStatus.Expiry) * time.Second)
		desiredAmount = lnStatus.Value
	}

	ctx, cancel := context.WithDeadline(c, fin)
	defer cancel()

	paymentStatus := make(chan *common.StatusReply)
	if len(hash) > 0 {
		go func() {
			paymentStatus <- checkLn(ctx, hash)
		}()
	}

	// keep polling for status update every N seconds
	if len(addr) > 0 {
		go func() {
			paymentStatus <- checkBtc(fin, addr, len(hash) > 0, desiredAmount)
		}()
	}

	// blocks until first channel message is received
	status := <-paymentStatus

	if status == nil {
		c.AbortWithStatusJSON(500, common.StatusReply{Error: "unknown error"})
		return
	}

	if status.Code < 300 {
		c.JSON(status.Code, status)
		return
	}

	c.AbortWithStatusJSON(status.Code, status)
}

// TODO: pagination
// TODO: limit
func history(c *gin.Context) {
	var queryParams struct {
		Limit      int64  `form:"limit"`
		Offset     int64  `form:"offset"`
		OnlyStatus string `form:"only_status" validate:"omitempty,oneof=paid expired pending"`
	}

	err := c.BindQuery(&queryParams)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	err = validator.New().Struct(queryParams)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	var warning string
	// fetch bitcoin history
	btcAllAddresses, err := btcClient.CheckAddress("")
	if err != nil {
		warning = "Unable to fetch Bitcoin history. Only showing LN."
	}

	// Convert Bitcoin history from list to easily addressable map
	btcHistory := make(map[string]common.AddrStatus)
	for _, payment := range btcAllAddresses {
		if payment.Label != "" {
			btcHistory[payment.Label] = payment
		}
	}

	// fetch LN history
	lnHistory, err := lnClient.History(c)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"error": fmt.Sprintf("Can't get history from LN node: %v", err),
		})
		return
	}

	// merge histories
	var history []common.Payment
	for _, invoice := range lnHistory {
		var payment common.Payment
		payment.ApplyLn(invoice)

		if btcStatus, ok := btcHistory[payment.Hash]; ok {
			payment.ApplyBtc(btcStatus)
		}

		switch queryParams.OnlyStatus {
		case "paid":
			if !payment.Paid {
				continue
			}

		case "expired":
			if !payment.Expired {
				continue
			}

		case "pending":
			if payment.Paid || payment.Expired {
				continue
			}
		}

		history = append(history, payment)
	}

	// reverse order before returning (newest on top)
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	c.JSON(200, struct {
		History []common.Payment `json:"history"`
		Error   string           `json:"error,omitempty"`
	}{
		History: history,
		Error:   warning,
	})
}

func info(c *gin.Context) {
	info, err := lnClient.Info(c)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"error": fmt.Sprintf("Can't get info from LN node: %v", err),
		})
		return
	}

	c.JSON(200, info.Uris)
}

func main() {
	//gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(cors.Default())
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	r := router.Group("/api")
	r.POST("/payment", newPayment)
	r.GET("/payment", status)
	r.GET("/info", info)

	// history only available if Basic Auth is enabled
	if len(accounts) > 0 {
		r.GET("/history", gin.BasicAuth(accounts), history)
	}

	if conf.StaticDir != "" {
		router.StaticFile("/", path.Join(conf.StaticDir, "index.html"))
	}

	if conf.Port == 0 {
		conf.Port = DefaultInvoicerPort
	}

	err := router.Run(fmt.Sprintf(":%d", conf.Port))
	if err != nil {
		panic(err)
	}
}
