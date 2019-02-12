package common

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type (
	Bitcoind struct {
		Host string `toml:"host"`
		Port int64  `toml:"port"`
		User string `toml:"user"`
		Pass string `toml:"pass"`
	}

	Lnd struct {
		Host      string `toml:"host"`
		Port      int64  `toml:"port"`
		Tls       string `toml:"tls"`
		Macaroons struct {
			Invoice  string `toml:"invoice"`
			ReadOnly string `toml:"readonly"`
		} `toml:"macaroon"`
	}

	Config struct {
		Port      int64             `toml:"port"`
		StaticDir string            `toml:"static-dir"`
		LnClient  string            `toml:"ln-client"`
		Bitcoind  Bitcoind          `toml:"bitcoind"`
		Lnd       Lnd               `toml:"lnd"`
		Users     map[string]string `toml:"users"`
	}
)

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
