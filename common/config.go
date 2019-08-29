package common

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type (
	// This struct defines the structure of the config .toml file
	Config struct {
		// Port invoicer will run on
		Port int64 `toml:"port"`

		// Path to directory where `index.html` will be served from
		StaticDir string `toml:"static-dir"`

		// Location of a log file
		LogFile string `toml:"log-file"`

		// Currently only `lnd` supported
		LnClient string `toml:"ln-client"`

		// Allows for disabling the possibility of on-chain payments.
		OffChainOnly bool `toml:"off-chain-only"`

		// [bitcoind] section in the `--config` file that defines Bitcoind's setup
		Bitcoind Bitcoind `toml:"bitcoind"`

		// [lnd] section in the `--config` file that defines Lnd's setup
		Lnd Lnd `toml:"lnd"`

		// An optional list of user:password pairs that will get granted access to the /history endpoint
		Users map[string]string `toml:"users"`
	}

	// Bitcoind config
	// NOTE: Keep in mind that this is **not yet encrypted**, so best to keep it _local_
	Bitcoind struct {
		Host string `toml:"host"`
		Port int64  `toml:"port"`
		User string `toml:"user"`
		Pass string `toml:"pass"`
	}

	// Lnd config
	Lnd struct {
		Host      string `toml:"host"`
		Port      int64  `toml:"port"`

		// TLS certificate is usually located in `~/.lnd/tls.cert`
		Tls       string `toml:"tls"`

		// Macaroons are usually located in `~/.lnd/data/chain/bitcoin/mainnet/`
		Macaroons struct {
			// This is needed to generate new invoices
			Invoice  string `toml:"invoice"`

			// This is needed to check status of invoices (and if enabled access `/history` endpoint)
			ReadOnly string `toml:"readonly"`
		} `toml:"macaroon"`

		// How many times try to talk to LND before committing suicide
		KillCount *int `toml:"kill-count"`
	}
)

// CleanAndExpandPath converts passed file system paths into absolute ones.
func CleanAndExpandPath(path string) string {
	if path == "" {
		return ""
	}

	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		var homeDir string
		u, err := user.Current()
		if err == nil {
			homeDir = u.HomeDir
		} else {
			homeDir = os.Getenv("HOME")
		}

		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but the variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}
