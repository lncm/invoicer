package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/lncm/invoicer/bitcoind"
	"github.com/lncm/invoicer/clightning"
	"github.com/lncm/invoicer/common"
	"github.com/lncm/invoicer/lnd"
	"github.com/pkg/errors"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

type (
	BitcoinWallet interface {
		Address(bech32 bool) (string, error)
	}

	BitcoinClient interface {
		BitcoinWallet

		BlockCount() (int64, error)
		ImportAddress(address, label string) error
		CheckAddress(address string) (common.AddrsStatus, error)
	}

	LightningClient interface {
		BitcoinWallet

		Info() (common.Info, error)
		Invoice(amount int64, desc string) (common.Invoice, error)
		Status(hash string) (common.Status, error)
		History() (common.Invoices, error)
	}
)

var (
	version,
	gitHash string

	lnClient  LightningClient
	btcClient BitcoinClient

	usersFile    = flag.String("users-file", "", "path to a file with acceptable user passwords")
	lnClientName = flag.String("ln-client", lnd.ClientName, "specify which LN implementation should be used. Allowed: lnd and clightning")

	indexFile = flag.String("index-file", "static/index.html", "pass path to a default index file")
	staticDir = flag.String("static-dir", "", "pass path to a dir containing static files to be served")
	port      = flag.Int64("port", 8080, "specify port to serve the website & API at")

	accounts gin.Accounts
)

func init() {
	flag.Parse()

	// init specified LN client
	switch strings.ToLower(*lnClientName) {
	case lnd.ClientName:
		lnClient = lnd.New()

	case cLightning.ClientName:
		lnClient = cLightning.New()

	default:
		panic("invalid LN client specified")
	}

	// init  BTC client for monitoring on-chain payments
	btcClient = bitcoind.New()

	versionString := "debug"
	if version != "" && gitHash != "" {
		versionString = fmt.Sprintf("%s (git: %s)", version, gitHash)
	}

	fmt.Printf("version: %s\nLN client: %s\n\n", versionString, *lnClientName)

	if usersFile != nil && len(*usersFile) > 0 {
		f, err := os.Open(*usersFile)
		if err != nil {
			fmt.Printf("Error: list of users for Basic Authentication not found at %s\n\n", *usersFile)
			fmt.Printf("Create a file (%s) in a format of:\n\n<user1> <password>\n<user2> <password>\n\nOr specify different path to the file using --users-file= flag\n", common.DefaultUsersFile)
			os.Exit(1)
		}

		defer func() {
			err = f.Close()
			if err != nil {
				fmt.Println(errors.Wrap(err, "error closing users file"))
			}
		}()

		accounts = make(gin.Accounts)

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			rawLine := scanner.Text()
			line := strings.Split(rawLine, " ")

			if len(line) != 2 || len(line[0]) == 0 || len(line[1]) == 0 {
				fmt.Printf("Error: can't read list of users for Basic Authentication from %s\n", *usersFile)
				fmt.Printf("Error found in line: \"%s\"\n\n", rawLine)
				fmt.Printf("Create a file (%s) in a format of:\n\n<user1> <password>\n<user2> <password>\n\nOr specify different path to the file using --users-file= flag\n", common.DefaultUsersFile)
				os.Exit(1)
			}

			accounts[line[0]] = line[1]
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error: can't read file %s: %v", *usersFile, err)
			os.Exit(1)
		}
	}
}

func newPayment(c *gin.Context) {
	var data struct {
		Amount      int64  `json:"amount"`
		Description string `json:"desc"`
	}

	err := c.ShouldBindJSON(&data)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	if len(data.Description) > common.MaxInvoiceDescLen {
		c.AbortWithStatusJSON(400, gin.H{
			"error": fmt.Sprintf("description too long. Max length is %d.", common.MaxInvoiceDescLen),
		})
		return
	}

	var payment common.NewPayment

	// Generate new LN invoice
	newInvoice, err := lnClient.Invoice(data.Amount, data.Description)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"error": errors.WithMessage(err, "can't create new LN invoice").Error(),
		})
		return
	}
	payment.Hash = newInvoice.Hash
	payment.Bolt11 = newInvoice.Bolt11

	// Extract invoice's creation date & expiry
	invoice, err := lnClient.Status(newInvoice.Hash)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"error": errors.WithMessage(err, "can't get LN invoice").Error(),
		})
		return
	}
	payment.CreatedAt = invoice.Ts
	payment.Expiry = invoice.Expiry

	// get BTC address
	payment.Address, err = lnClient.Address(false)
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

	c.JSON(200, payment)
}

func lnStatus(c *gin.Context) {
	hash := c.Param("hash")
	if len(hash) == 0 {
		c.AbortWithStatusJSON(400, "err: preimage hash needs to be provided")
		return
	}

	fin := time.Now().Add(common.DefaultInvoiceExpiry * time.Second)
	for time.Now().Before(fin) {
		status, err := lnClient.Status(hash)
		if err != nil {
			c.AbortWithStatusJSON(400, fmt.Sprintf("err: %v", err))
			return
		}

		if status.Settled {
			c.JSON(200, "paid")
			return
		}

		if status.IsExpired() {
			c.AbortWithStatusJSON(408, "expired")
			return
		}

		time.Sleep(5 * time.Second)
	}

	c.AbortWithStatusJSON(408, "expired")
}

func getAmount(hash string) int64 {
	// verify label is a hex number (LN hash)
	re := regexp.MustCompile("^[[:xdigit:]]{64}$")
	if !re.MatchString(hash) {
		return -1 // error: no known invoice to get the data from
	}

	s, err := lnClient.Status(hash)
	if err != nil {
		log.Println(err)
		return -1 // error: unable to get the invoice
	}

	return s.Value

}

func btcStatus(c *gin.Context) {
	addr := c.Param("address")
	if len(addr) == 0 {
		c.AbortWithStatusJSON(400, "err: address needs to be provided")
		return
	}

	var desiredAmount int64

	fin := time.Now().Add(common.DefaultInvoiceExpiry * time.Second)
	for time.Now().Before(fin) {
		statuses, err := btcClient.CheckAddress(addr)
		if err != nil {
			c.AbortWithStatusJSON(400, fmt.Sprintf("err: %v", err))
			return
		}

		status := statuses[0]

		if desiredAmount == 0 {
			desiredAmount = getAmount(status.Label)
		}

		receivedAmount := int64(status.Amount * 1e8)
		if receivedAmount > 0 {
			if desiredAmount == receivedAmount {
				c.JSON(200, fmt.Sprintf("[%d conf] paid", status.Confirmations))
				return
			}

			if receivedAmount > desiredAmount {
				c.JSON(202, fmt.Sprintf("[%d conf] over-paid by %d sat", status.Confirmations, receivedAmount-desiredAmount))
				return
			}

			if desiredAmount > receivedAmount {
				c.AbortWithStatusJSON(402, fmt.Sprintf("err: [%d conf] under-paid by %d sat", status.Confirmations, desiredAmount-receivedAmount))
				return
			}
		}

		time.Sleep(5 * time.Second)
	}

	c.JSON(408, "expired")
}

// TODO: pagination
// TODO: only paid
func history(c *gin.Context) {
	history, err := lnClient.History()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"error": fmt.Sprintf("Can't get history from LN node: %v", err),
		})
		return
	}

	c.JSON(200, history)
}

func info(c *gin.Context) {
	info, err := lnClient.Info()
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

	r := &router.RouterGroup
	if len(accounts) > 0 {
		r = router.Group("/", gin.BasicAuth(accounts))
	} else if len(*usersFile) != 0 {
		panic("users.list passed, but no accounts detected")
	}

	r.StaticFile("/", *indexFile)

	if *staticDir != "" {
		r.Static("/static/", *staticDir)
	}

	r.POST("/payment", newPayment)
	r.GET("/payment", newPayment)             // TODO: remove; only here for testing
	r.GET("/payment/ln/:hash", lnStatus)      // TODO: change reply format
	r.GET("/payment/btc/:address", btcStatus) // TODO: change reply format
	r.GET("/info", info)

	// TODO: only behind auth
	if len(accounts) > 0 {
		r.GET("/history", history)
	}

	err := router.Run(fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
}
