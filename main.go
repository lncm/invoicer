package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	defaultUsersFile = "./users.list"

	maxDescLen    = 639
	invoiceExpiry = 180
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

// BEGIN: Structs for Reading JSON for C lightning Files
// Network Info
type NetworkInfoStructure struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}
type lightningGetInfoStructure struct {
	Id          string                 `json:"id"`
	Alias       string                 `json:"alias"`
	Version     string                 `json:"version"`
	Network     string                 `json:"network"`
	BlockHeight int                    `json:"blockheight"`
	Address     []NetworkInfoStructure `json:"address"`
	Binding     []NetworkInfoStructure `json:"binding"`
}

// Helper function for c lightning structs
func (resultToRead *lightningGetInfoStructure) read(json_code string) {
	if e := json.Unmarshal([]byte(json_code), resultToRead); e != nil {
		fmt.Printf("ERROR JSON decode: %v", e)
	}
}

// END: Structs for Reading JSON for C lightning Files
// BEGIN: Functions for C-lightning
func clightningconnstring(cmd_response string) (connstring string) {
	var info lightningGetInfoStructure

	info.read(cmd_response)
	if len(info.Address) > 0 {
		if len(info.Address) == 1 {
			return fmt.Sprintf("%s@%s:%d", info.Id, info.Address[0].Address, info.Address[0].Port)
		} else {
			// TODO: Return string of addresses
			return fmt.Sprintf("%s@%s:%d", info.Id, info.Address[0].Address, info.Address[0].Port)
		}
	} else {
		return fmt.Sprintf("%s", info.Id)
	}
}

// END: Functions for C-lightning

func (s Status) IsExpired() bool {
	return time.Now().After(time.Unix(s.Ts+s.Expiry, 0))
}

var (
	usersFile   = flag.String("users-file", defaultUsersFile, "path to a file with acceptable user passwords")
	lncliBinary = flag.String("lncli-binary", "/usr/local/bin/lncli", "Specify custom path to lncli binary")
	mainnet     = flag.Bool("mainnet", false, "Set to true if this node will run on mainnet")

	accounts gin.Accounts
	network  = "testnet"
)

func init() {
	flag.Parse()

	if *mainnet {
		network = "mainnet"
	}

	fmt.Printf(" binary:\t%s\nmainnet:\t%t\n  users:\t%s\n\n", *lncliBinary, *mainnet, *usersFile)

	f, err := os.Open(*usersFile)
	if err != nil {
		fmt.Printf("Error: list of users for Basic Authentication not found at %s\n\n", *usersFile)
		fmt.Printf("Create a file (%s) in a format of:\n\n<user1> <password>\n<user2> <password>\n\nOr specify different path to the file using --users-file= flag\n", defaultUsersFile)
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
			fmt.Printf("Create a file (%s) in a format of:\n\n<user1> <password>\n<user2> <password>\n\nOr specify different path to the file using --users-file= flag\n", defaultUsersFile)
			os.Exit(1)
		}

		accounts[line[0]] = line[1]
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error: can't read file %s: %v", *usersFile, err)
		os.Exit(1)
	}

}

func getInvoice(amount float64, desc string) (invoice Invoice, err error) {
	cmd := exec.Command(
		*lncliBinary,
		fmt.Sprintf("--network=%s", network),
		"addinvoice",
		fmt.Sprintf("--expiry=%d", invoiceExpiry), // TODO: allow for custom expiry on invoices
		fmt.Sprintf("--memo=%s", desc),            // TODO: sanitize `desc` better
		fmt.Sprintf("%d", int(amount)),
	)

	var out, err2 bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err2

	err = cmd.Run()
	if err != nil {
		return invoice, errors.Wrap(err, err2.String())
	}

	err = json.NewDecoder(&out).Decode(&invoice)
	if err != nil {
		return invoice, errors.Wrap(err, "unable to decode response")
	}

	return
}

func getStatus(hash string) (s Status, err error) {
	cmd := exec.Command(
		*lncliBinary,
		fmt.Sprintf("--network=%s", network),
		"lookupinvoice",
		hash,
	)

	var out, err2 bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err2

	err = cmd.Run()
	if err != nil {
		return s, errors.Wrap(err, err2.String())
	}

	err = json.NewDecoder(&out).Decode(&s)
	if err != nil {
		return s, errors.Wrap(err, "unable to decode response")
	}

	return s, nil
}

func getInfo() (info Info, err error) {
	cmd := exec.Command(
		*lncliBinary,
		fmt.Sprintf("--network=%s", network),
		"getinfo",
	)

	var out, err2 bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &err2

	err = cmd.Run()
	if err != nil {
		return info, errors.Wrap(err, err2.String())
	}

	err = json.NewDecoder(&out).Decode(&info)
	if err != nil {
		return info, errors.Wrap(err, "unable to decode response")
	}

	return info, nil
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
	if len(desc) > maxDescLen {
		c.JSON(400, gin.H{
			"error": fmt.Sprintf("Description too long. Max length is %d.", maxDescLen),
		})
		return
	}

	invoice, err := getInvoice(amount, desc)
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

	fin := time.Now().Add(invoiceExpiry * time.Second)
	for time.Now().Before(fin) {
		status, err := getStatus(hash)
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
	info, err := getInfo()
	if err != nil {
		// Try C lightning too
		clightoutput, clighterr := exec.Command("/usr/bin/docker", "exec", "lightningpay", "lightning-cli", "getinfo").Output()
		connstring := clightningconnstring(fmt.Sprintf("%s", clightoutput))
		if clighterr != nil {
			c.JSON(500, gin.H{
				"error": fmt.Sprintf("Can't get info from LND node: %v", err),
			})
			return
		} else {
			c.String(200, fmt.Sprintf("{\"Uris\": [\"%s\"]}", connstring))
			return
		}
	}

	c.JSON(200, info.Uris)
}

// Initial C Lightning Function
func clightninginfo(c *gin.Context) {
	out, err := exec.Command("/usr/bin/docker", "exec", "lightningpay", "lightning-cli", "getinfo").Output()
	if err == nil {
		c.String(200, fmt.Sprintf("%s", out))
	} else {
		c.JSON(500, gin.H{
			"error": fmt.Sprintf("Error from lightning service: %s", err),
		})
		return
	}
}

func main() {
	//gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	authorized := r.Group("/", gin.BasicAuth(accounts))

	authorized.GET("/invoice", invoice)
	authorized.GET("/status/:hash", status)
	authorized.GET("/connstrings", info)
	authorized.GET("/clightning-info", clightninginfo) // runs getinfo

	r.Run(":1666")
}
