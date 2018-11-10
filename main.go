package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lncm/invoicer/clightning"
	"github.com/lncm/invoicer/common"
	"github.com/lncm/invoicer/docker-clightning"
	"github.com/lncm/invoicer/lnd"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LnClient interface {
	Info() (common.Info, error)
	Invoice(amount float64, desc string) (common.Invoice, error)
	Status(hash string) (common.Status, error)
}

var client LnClient

var (
	usersFile = flag.String("users-file", common.DefaultUsersFile, "path to a file with acceptable user passwords")
	noAuth    = flag.Bool("no-auth", false, "set to make endpoint not require auth ")
	lnClient  = flag.String("ln-client", lnd.ClientName, "specify which LN implementation should be used. Allowed: lnd, clightning, docker-clightning")
	lnBinary  = flag.String("ln-binary", "/usr/local/bin/lncli", "Specify custom path to the LN instance binary binary")
	// NOTE: lncli-binary -> ln-binary & tell @AnotherDroog about this breaking change
	mainnet = flag.Bool("mainnet", false, "Set to true if this node will run on mainnet")

	accounts gin.Accounts
	network  = "testnet"
)

func init() {
	flag.Parse()

	if *mainnet {
		network = "mainnet"
	}

	switch strings.ToLower(*lnClient) {
	case lnd.ClientName:
		client = lnd.New(*lnBinary, network)

	case cLightning.ClientName:
		client = cLightning.New(*lnBinary, network)

	case dockerCLightning.ClientName:
		client = dockerCLightning.New(*lnBinary, network)

	default:
		panic("invalid client specified")
	}

	fmt.Printf(" binary:\t%s\nmainnet:\t%t\nclient:\t%s\n  users:\t%s\n\n", *lnBinary, *mainnet, *lnClient, *usersFile)

	if !*noAuth {
		f, err := os.Open(*usersFile)
		if err != nil {
			fmt.Printf("Error: list of users for Basic Authentication not found at %s\n\n", *usersFile)
			fmt.Printf("Create a file (%s) in a format of:\n\n<user1> <password>\n<user2> <password>\n\nOr specify different path to the file using --users-file= flag\n", common.DefaultUsersFile)
			os.Exit(1)
		}

		defer f.Close()

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

func invoice(c *gin.Context) {
	rawAmount := c.DefaultQuery("amount", "0")
	amount, err := strconv.ParseFloat(rawAmount, 64)
	if err != nil {
		c.JSON(400, gin.H{
			"error": "Can't parse amount. Make sure it's in satoshis.",
		})
		return
	}

	desc := c.DefaultQuery("desc", "")
	if len(desc) > common.MaxDescLen {
		c.JSON(400, gin.H{
			"error": fmt.Sprintf("Description too long. Max length is %d.", common.MaxDescLen),
		})
		return
	}

	invoice, err := client.Invoice(amount, desc)
	if err != nil {
		c.JSON(500, gin.H{
			"error": fmt.Sprintf("Can't get invoice from LN node: %v", err),
		})
		return
	}

	c.JSON(200, gin.H{
		"invoice": invoice.Bolt11,
		"hash":    invoice.Hash,
	})
}

func status(c *gin.Context) {
	hash := c.Param("hash")

	fin := time.Now().Add(common.DefaultInvoiceExpiry * time.Second)
	for time.Now().Before(fin) {
		status, err := client.Status(hash)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}

		if status.Settled {
			c.JSON(200, "paid")
			return
		}

		if status.IsExpired() {
			c.JSON(408, "expired")
			return
		}

		time.Sleep(5 * time.Second)
	}

	c.JSON(408, "expired")
}

func info(c *gin.Context) {
	info, err := client.Info()
	if err != nil {
		c.JSON(500, gin.H{
			"error": fmt.Sprintf("Can't get info from LN node: %v", err),
		})
		return
	}

	c.JSON(200, info.Uris)
}

func main() {
	//gin.SetMode(gin.ReleaseMode)

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Use(cors.Default())

	if *noAuth {
		// TODO: will it work out of _the box_ with Basic Auth
		r.StaticFile("/", fmt.Sprintf("%s/index.html", dir))

		r.GET("/invoice", invoice)
		r.GET("/status/:hash", status)
		r.GET("/connstrings", info)
	} else {
		authorized := r.Group("/", gin.BasicAuth(accounts))

		authorized.GET("/invoice", invoice)
		authorized.GET("/status/:hash", status)
		authorized.GET("/connstrings", info)
	}

	//authorized.GET("/clightning-info", clightningInfo) // runs getinfo

	err = r.Run(":1666")
	if err != nil {
		panic(err)
	}
}
