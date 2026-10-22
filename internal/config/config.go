// Package config loads local settings without accepting remote inference endpoints.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Endpoint  string `toml:"endpoint"`
	DataDir   string `toml:"data_dir"`
	Root      string `toml:"root"`
	Mode      string `toml:"mode"`
	Context   int    `toml:"context"`
	MaxOutput int    `toml:"max_output"`
}

func Default() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(home, ".local", "share")
	}
	root, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}
	return Config{Endpoint: "http://127.0.0.1:8081", DataDir: filepath.Join(base, "dotamanai-codex"), Root: root, Mode: "analysis", Context: 4096, MaxOutput: 512}, nil
}
func ValidateEndpoint(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "http" || ip == nil || !ip.IsLoopback() || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("endpoint must be numeric HTTP loopback without credentials, path or query")
	}
	return nil
}
func (c Config) Validate() error {
	if err := ValidateEndpoint(c.Endpoint); err != nil {
		return err
	}
	if !filepath.IsAbs(c.DataDir) || !filepath.IsAbs(c.Root) {
		return fmt.Errorf("data_dir and root must be absolute")
	}
	if c.Mode != "analysis" && c.Mode != "confirm" && c.Mode != "auto" {
		return fmt.Errorf("invalid mode")
	}
	if c.Context < 512 || c.Context > 131072 || c.MaxOutput < 1 || c.MaxOutput >= c.Context/2 {
		return fmt.Errorf("invalid context budget")
	}
	return nil
}
func Load(path string) (Config, error) {
	c, err := Default()
	if err != nil {
		return c, err
	}
	if path != "" {
		f, e := os.Open(path)
		if e != nil {
			return c, e
		}
		defer f.Close()
		d := toml.NewDecoder(f).DisallowUnknownFields()
		if e = d.Decode(&c); e != nil {
			return c, fmt.Errorf("decode config: %w", e)
		}
	}
	c.Endpoint = strings.TrimRight(c.Endpoint, "/")
	return c, c.Validate()
}
