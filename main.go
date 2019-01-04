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
	Invoice(amount float64, desc string) (common.NewPayment, error)
	Status(hash string) (common.Status, error)
	History() (common.Invoices, error)
}

var (
	client  LnClient
	version,
	gitHash string

	usersFile = flag.String("users-file", "", "path to a file with acceptable user passwords")
	lnClient  = flag.String("ln-client", lnd.ClientName, "specify which LN implementation should be used. Allowed: lnd, and clightning")

	indexFile = flag.String("index-file", "static/index.html", "pass path to a default index file")
	staticDir = flag.String("static-dir", "", "pass path to a dir containing static files to be served")
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

	versionString := "debug"
	if version != "" && gitHash != "" {
		versionString = fmt.Sprintf("%s (git: %s)", version, gitHash)
	}

	fmt.Printf("version: %s\n client: %s\n\n", versionString, *lnClient)

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

	c.JSON(200, invoice)
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

func history(c *gin.Context) {
	history, err := client.History()
	if err != nil {
		c.JSON(500, gin.H{
			"error": fmt.Sprintf("Can't get history from LN node: %v", err),
		})
		return
	}

	c.JSON(200, history)
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

	router := gin.Default()
	router.Use(cors.Default())

	// merchant flow (run everything behind basic auth)
	if len(accounts) > 0 {
		authorized := router.Group("/", gin.BasicAuth(accounts))

		authorized.GET("/payment", invoice)
		authorized.GET("/payment/:hash", status)
		authorized.GET("/info", info)

		// donations flow (run everything without extra auth)
	} else if len(*usersFile) == 0 {
		router.StaticFile("/", *indexFile)

		if *staticDir != "" {
			router.Static("/static/", *staticDir)
		}

		router.GET("/payment", invoice)
		router.GET("/payment/:hash", status)
		router.GET("/info", info)

		// TODO: only behind auth
		router.GET("/history", history)

	} else {
		panic("users.list passed, but no accounts detected")
	}

	err := router.Run(fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
}
