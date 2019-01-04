package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lncm/invoicer/clightning"
	"github.com/lncm/invoicer/common"
	"github.com/lncm/invoicer/lnd"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type LnClient interface {
	Info() (common.Info, error)
	Address() (string, error)
	Invoice(amount float64, desc string) (common.Invoice, error)
	Status(hash string) (common.Status, error)
}

var (
	client  LnClient
	version,
	gitHash string

	usersFile = flag.String("users-file", "", "path to a file with acceptable user passwords")
	lnClient  = flag.String("ln-client", lnd.ClientName, "specify which LN implementation should be used. Allowed: lnd, clightning, docker-clightning")

	indexFile = flag.String("index-file", "index.html", "pass path to a default index file")
	port      = flag.Int64("port", 1666, "specify port to serve the website & API at")

	accounts gin.Accounts
)

func init() {
	flag.Parse()

	switch strings.ToLower(*lnClient) {
	case lnd.ClientName:
		client = lnd.New()

	case cLightning.ClientName:
		client = cLightning.New()

	default:
		panic("invalid client specified")
	}

	fmt.Printf("version: %s (git: %s)\n client: %s\n\n", version, gitHash, *lnClient)

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

	r := gin.Default()
	r.Use(cors.Default())

	// run everything behind basic auth
	if len(accounts) > 0 {
		authorized := r.Group("/", gin.BasicAuth(accounts))

		authorized.GET("/invoice", invoice)
		authorized.GET("/status/:hash", status)
		authorized.GET("/connstrings", info)

		// run everything without extra auth
	} else if len(*usersFile) == 0 {
		r.StaticFile("/", *indexFile)

		r.GET("/invoice", invoice)
		r.GET("/status/:hash", status)
		r.GET("/connstrings", info)

	} else {
		panic("users.list passed, but no accounts detected")
	}

	err := r.Run(fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
}
