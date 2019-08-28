package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/lncm/invoicer/bitcoind"
	"github.com/lncm/invoicer/clightning"
	"github.com/lncm/invoicer/common"
	"github.com/lncm/invoicer/lnd"
)

type (
	BitcoinClient interface {
		Address(bech32 bool) (string, error)
		BlockCount() (int64, error)
		ImportAddress(address, label string) error
		CheckAddress(address string) (common.AddrsStatus, error)
	}

	LightningClient interface {
		NewAddress(ctx context.Context, bech32 bool) (string, error)
		Info(ctx context.Context) (common.Info, error)
		NewInvoice(ctx context.Context, amount int64, desc string) (string, string, error)
		Status(ctx context.Context, hash string) (common.Status, error)
		StatusWait(ctx context.Context, hash string) (common.Status, error)
		History(ctx context.Context) (common.Invoices, error)
	}

	lnStatusFn func(c context.Context, hash string) (common.Status, error)

	CustomValidator struct {
		validator *validator.Validate
	}
)

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

const DefaultInvoicerPort = 8080

var (
	version,
	gitHash string

	lnClient  LightningClient
	btcClient BitcoinClient
	conf      common.Config

	configFilePath = flag.String("config", common.DefaultConfigFile, "Path to a config file in TOML format")
	showVersion    = flag.Bool("version", false, "Show version and exit")
)

func init() {
	flag.Parse()

	versionString := "debug"
	if version != "" && gitHash != "" {
		versionString = fmt.Sprintf("%s (git: %s)", version, gitHash)
	}

	// if `--version` flag set, just show the version, and exit
	if *showVersion {
		fmt.Println(versionString)
		os.Exit(0)
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Expand configFile file path and load it
	configFile, err := toml.LoadFile(common.CleanAndExpandPath(*configFilePath))
	if err != nil {
		configFile, err = common.DeprecatedConfigLocationCheck(*configFilePath, err)
		if err != nil {
			panic(xerrors.Errorf("unable to load %s:\n\t%w", *configFilePath, err))
		}

		log.Warningln("WARNING: Default config location (~/.invoicer/) has been changed to ~/.lncm/ !\n" +
			"\tPlease rename it, as future versions will no longer check for the config file there.")
	}

	// Try to understand the config file
	err = configFile.Unmarshal(&conf)
	if err != nil {
		panic(xerrors.Errorf("unable to process %s:\n\t%w", *configFilePath, err))
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
		// lnClient = cLightning.New()

	default:
		panic(xerrors.Errorf("invalid ln-client specified: %s", conf.LnClient))
	}

	// Init BTC client for monitoring on-chain payments
	btcClient, err = bitcoind.New(conf.Bitcoind)
	if err != nil {
		panic(err)
	}

	if conf.LogFile == "" {
		conf.LogFile = common.DefaultLogFile
	}

	fields := log.Fields{
		"version":   versionString,
		"client":    conf.LnClient,
		"users":     len(conf.Users),
		"conf-file": *configFilePath,
		"log-file":  conf.LogFile,
	}

	// Write current config to stdout
	log.WithFields(fields).Println("invoicer started")

	// After all initialization has been done, start logging to log file
	log.SetOutput(&lumberjack.Logger{
		Filename:  common.CleanAndExpandPath(conf.LogFile),
		LocalTime: true,
		Compress:  true,
	})
	log.SetFormatter(&log.JSONFormatter{
		PrettyPrint: false, // Having `false` here makes sure that `jq` always works on `tail -f`.
	})

	// Write current config to log file
	log.WithFields(fields).Println("invoicer started")
}

func newPayment(c echo.Context) error {
	var data struct {
		Amount      int64  `json:"amount"`
		Description string `json:"desc"`
		Only        string `json:"only"`
	}

	err := c.Bind(&data)
	if err != nil {
		return c.JSON(400, errorStatus(err.Error()))
	}

	if data.Only != "" && data.Only != "btc" && data.Only != "ln" {
		return c.JSON(400, errorStatus("only= is an optional parameter that can only take `btc` and `ln` as values"))
	}

	var payment common.NewPayment

	if data.Only != "btc" {
		if len(data.Description) > common.MaxInvoiceDescLen {
			return c.JSON(400, errorStatus("description too long. Max length is %d", common.MaxInvoiceDescLen))
		}

		// Generate new LN invoice
		payment.Bolt11, payment.Hash, err = lnClient.NewInvoice(c.Request().Context(), data.Amount, data.Description)
		if err != nil {
			return c.JSON(500, errorStatus("can't create new LN invoice: %w", err))
		}

		// Extract invoice's creation date & expiry
		invoice, err := lnClient.Status(c.Request().Context(), payment.Hash)
		if err != nil {
			return c.JSON(500, errorStatus("can't get LN invoice: %w", err))
		}
		payment.CreatedAt = invoice.Ts
		payment.Expiry = invoice.Expiry
	}

	if data.Only != "ln" {
		// get BTC address
		payment.Address, err = lnClient.NewAddress(c.Request().Context(), false)
		if err != nil {
			return c.JSON(500, errorStatus("can't get Bitcoin address: %w", err))
		}

		label := data.Description
		if len(payment.Hash) > 0 {
			label = payment.Hash
		}

		err = btcClient.ImportAddress(payment.Address, label)
		if err != nil {
			return c.JSON(500, errorStatus("can't import address (%s) to Bitcoin node: %w", payment.Address, err))
		}
	}

	log.WithFields(log.Fields{
		"in":  data,
		"out": payment,
	}).Println("Payment requested")

	return c.JSON(200, payment)
}

func checkLnStatus(c context.Context, hash string, statusFn lnStatusFn) *common.StatusReply {
	status, err := statusFn(c, hash)
	if err != nil {
		return &common.StatusReply{
			Code:  500,
			Error: fmt.Sprintf("unable to fetch invoice: %s", err),
		}
	}

	if status.Settled {
		return &common.StatusReply{
			Code: 200, Ln: &status,
		}
	}

	if status.IsExpired() {
		return &common.StatusReply{
			Code:  408,
			Error: "expired",
		}
	}

	return &common.StatusReply{Ln: &status}
}

func checkBtcStatus(ctx context.Context, fin time.Time, addr string, lnProvided, flexible bool, desiredAmount int64) *common.StatusReply {
	for time.Now().Before(fin) {
		time.Sleep(2 * time.Second)

		if ctx.Err() != nil {
			return &common.StatusReply{
				Error: ctx.Err().Error(),
			}
		}

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

		if flexible || desiredAmount == receivedAmount {
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

func errorStatus(msg string, err ...interface{}) common.StatusReply {
	return common.StatusReply{
		Error: xerrors.Errorf(msg, err...).Error(),
	}
}

func status(c echo.Context) error {
	// var queryParams struct {
	// 	Hash     string `form:"hash"`
	// 	Addr     string `form:"address"`
	// 	Flexible bool   `form:"flexible"`
	// }
	//
	// err := c.BindQuery(&queryParams)
	// if err != nil {
	// 	return c.JSON(400, errorStatus("invalid request: %w", err))
	// }

	hash := c.QueryParam("hash")
	addr := c.QueryParam("address")
	// flexible := c.QueryParam("flexible")

	if len(hash) == 0 && len(addr) == 0 {
		return c.JSON(500, errorStatus("At least one of: `hash` or `address` needs to be provided"))
	}

	var desiredAmount int64
	fin := time.Now().Add(common.DefaultInvoiceExpiry * time.Second)

	// do initial LN invoice check, and adjust expiration if available
	if len(hash) > 0 {
		status := checkLnStatus(c.Request().Context(), hash, lnClient.Status)
		if status.Code > 0 {
			return c.JSON(status.Code, *status)
		}

		fin = time.Unix(status.Ln.Ts, 0).Add(time.Duration(status.Ln.Expiry) * time.Second)
		desiredAmount = status.Ln.Value
	}

	ctx, cancel := context.WithDeadline(c.Request().Context(), fin)
	defer cancel()

	paymentStatus := make(chan *common.StatusReply)

	// subscribe to LN invoice status changes
	if len(hash) > 0 {
		go func() {
			paymentStatus <- checkLnStatus(ctx, hash, lnClient.StatusWait)
		}()
	}

	// keep polling for status update every N seconds
	if len(addr) > 0 {
		go func() {
			paymentStatus <- checkBtcStatus(ctx, fin, addr, len(hash) > 0, true /*flexible*/, desiredAmount)
		}()
	}

	var status *common.StatusReply

	// wait until either:
	select {
	// … payment is received successfully
	case status = <-paymentStatus:

	// … payment expires
	case <-ctx.Done():
		status = &common.StatusReply{
			Code:  408,
			Error: "expired",
		}

	// … payment is cancelled by user
	case <-c.Request().Context().Done():
		cancel()
		status = &common.StatusReply{
			Code:  499,
			Error: "cancelled by client",
		}
	}

	log.WithFields(log.Fields{
		"in":     c.QueryParams(),
		"status": *status,
	}).Println("Payment updated")

	return c.JSON(status.Code, *status)
}

// TODO: pagination
// TODO: limit
// TODO: bitcoin transactions
func history(c echo.Context) error {
	var queryParams struct {
		Limit      int64  `form:"limit"`
		Offset     int64  `form:"offset"`
		OnlyStatus string `form:"only_status" validate:"omitempty,oneof=paid expired pending"`
	}

	err := c.Bind(&queryParams)
	if err != nil {
		return c.JSON(400, errorStatus("invalid request: %w", err))
	}

	err = validator.New().Struct(queryParams)
	if err != nil {
		return c.JSON(400, errorStatus("invalid request: %w", err))
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
	lnHistory, err := lnClient.History(c.Request().Context())
	if err != nil {
		return c.JSON(500, errorStatus("Can't get history from LN node: %w", err))
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

	return c.JSON(200, struct {
		History []common.Payment `json:"history"`
		Error   string           `json:"error,omitempty"`
	}{
		History: history,
		Error:   warning,
	})
}

func info(c echo.Context) error {
	info, err := lnClient.Info(c.Request().Context())
	if err != nil {
		return c.JSON(500, errorStatus("Can't get info from LN node: %w", err))
	}

	return c.JSON(200, info.Uris)
}

func main() {
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Gzip())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	api := e.Group("/api")
	api.POST("/payment", newPayment)
	api.GET("/payment", status)
	api.GET("/info", info)

	// history only available if Basic Auth is enabled
	if len(conf.Users) > 0 {
		e.Validator = &CustomValidator{validator: validator.New()}
		api.GET("/history", history, middleware.BasicAuth(func(user, pass string, c echo.Context) (bool, error) {
			p, ok := conf.Users[user]
			if !ok {
				return false, nil
			}

			return p == pass, nil
		}))
	}

	var staticFilePath string
	if conf.StaticDir != "" {
		staticFilePath = path.Join(conf.StaticDir, "index.html")
		e.File("/", staticFilePath)
	}

	if conf.Port == 0 {
		conf.Port = DefaultInvoicerPort
	}

	log.WithFields(log.Fields{
		"routes":      e.Routes(),
		"port":        conf.Port,
		"static-file": staticFilePath,
	}).Println("gin router defined")

	e.HideBanner = true

	// Start server
	go func() {
		if err := e.Start(fmt.Sprintf(":%d", conf.Port)); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
