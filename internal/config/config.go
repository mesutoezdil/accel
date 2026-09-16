// Package config loads accel's YAML configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Config is the file at `~/.config/accel/config.yaml` (or `--config`).
type Config struct {
	Refresh time.Duration `yaml:"refresh"` // sampling interval
	Vendors []string      `yaml:"vendors"` // empty: probe every vendor

	History struct {
		Keep       time.Duration `yaml:"keep"`        // how far back the time machine goes
		Resolution time.Duration `yaml:"resolution"`  // one point per device per resolution
		Dir        string        `yaml:"dir"`         // default: state dir
		Persist    bool          `yaml:"persist"`     // write history to disk
		MaxDiskMB  int           `yaml:"max_disk_mb"` // oldest files go first
	} `yaml:"history"`

	Thresholds struct {
		IdleUtil     float64       `yaml:"idle_util"`     // percent; below it a device is idle
		IdleAfter    time.Duration `yaml:"idle_after"`    // idle with processes this long: idle-allocated
		BusyUtil     float64       `yaml:"busy_util"`     // percent; at or above it a device is busy
		TempWarn     float64       `yaml:"temp_warn"`     // celsius
		OutlierDelta float64       `yaml:"outlier_delta"` // percentage points below siblings
	} `yaml:"thresholds"`

	Listen string `yaml:"listen"` // ":9800" serves the API and /metrics
	// Token is the bearer token the API requires, or its SHA-256 digest
	// prefixed with "sha256:" so the plaintext never sits in a file.
	Token string `yaml:"token"`
	TLS   struct {
		Cert     string `yaml:"cert"`      // server certificate (PEM)
		Key      string `yaml:"key"`       // server key (PEM)
		ClientCA string `yaml:"client_ca"` // when set, clients must present a certificate signed by it
	} `yaml:"tls"`
	// Insecure allows `--listen` outside loopback without TLS and a token.
	Insecure bool `yaml:"insecure"`

	Nodes []Node `yaml:"nodes"` // remote accel services or SSH hosts in the fleet

	Kubernetes struct {
		Kubeconfig string `yaml:"kubeconfig"` // outside the cluster; "" uses in-cluster or ~/.kube/config
		Context    string `yaml:"context"`
	} `yaml:"kubernetes"`

	Cost struct {
		Currency string             `yaml:"currency"` // "$"
		PerHour  map[string]float64 `yaml:"per_hour"` // device name substring: price per device-hour
	} `yaml:"cost"`

	Alerts struct {
		Webhook      string        `yaml:"webhook"`      // POST JSON on every new alert
		Slack        string        `yaml:"slack"`        // Slack incoming webhook URL
		Alertmanager string        `yaml:"alertmanager"` // http://alertmanager:9093
		MinSeverity  string        `yaml:"min_severity"` // info, warning, critical
		Resend       time.Duration `yaml:"resend"`       // repeat an active alert after this long
		Rules        []Rule        `yaml:"rules"`        // user conditions on top of the built-in ones
	} `yaml:"alerts"`

	Carbon struct {
		GramsPerKWh float64 `yaml:"g_per_kwh"` // grid intensity; 0 hides carbon
	} `yaml:"carbon"`

	Theme       string            `yaml:"theme"`       // built-in name or a file under ~/.config/accel/themes
	Colors      map[string]string `yaml:"colors"`      // role: color, overrides the theme
	Transparent bool              `yaml:"transparent"` // keep the terminal background
	Keys        map[string]string `yaml:"keys"`        // action: key
	Mouse       *bool             `yaml:"mouse"`       // default on
	Log         string            `yaml:"log"`         // debug log file
}

// Rule is a user alert: "<metric> <op> <value>" that must hold for a
// duration, optionally only on devices with processes or an allocation.
type Rule struct {
	Name     string        `yaml:"name"`
	When     string        `yaml:"when"`     // "util < 10", "temp >= 80", "mem_used > 70G", "health < 50"
	For      time.Duration `yaml:"for"`      // how long the condition must hold; 0 fires at once
	On       string        `yaml:"on"`       // "" any device, "allocated", "idle"
	Severity string        `yaml:"severity"` // info, warning, critical (default warning)
}

// Node is a remote source: an accel service (url) or an SSH host whose
// vendor tools accel runs itself (ssh).
type Node struct {
	Name      string `yaml:"name"`
	URL       string `yaml:"url"`
	Token     string `yaml:"token"`
	TokenFile string `yaml:"token_file"`
	CA        string `yaml:"ca"`  // CA file for the node's certificate
	SSH       string `yaml:"ssh"` // user@host[:port]
	Key       string `yaml:"key"` // private key for ssh (default: agent, ~/.ssh)
}

// Default is the configuration without a file.
func Default() Config {
	var c Config
	c.Refresh = time.Second
	c.History.Keep = 24 * time.Hour
	c.History.Resolution = 5 * time.Second
	c.History.Persist = true
	c.History.MaxDiskMB = 256
	c.Thresholds.IdleUtil = 5
	c.Thresholds.IdleAfter = 5 * time.Minute
	c.Thresholds.BusyUtil = 80
	c.Thresholds.TempWarn = 85
	c.Thresholds.OutlierDelta = 20
	c.Cost.Currency = "$"
	c.Alerts.MinSeverity = "warning"
	c.Alerts.Resend = time.Hour
	c.Theme = "default"
	return c
}

// Path is the default config file location.
func Path() string {
	if d := os.Getenv("ACCEL_CONFIG"); d != "" {
		return d
	}
	return filepath.Join(ConfigDir(), "config.yaml")
}

// ConfigDir is ~/.config/accel (XDG aware).
func ConfigDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "accel")
}

// StateDir is where history lives.
func StateDir() string {
	if d := os.Getenv("ACCEL_STATE_DIR"); d != "" {
		return d
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "accel")
}

// Load reads path over the defaults. A missing default file is fine; a
// missing explicit file is an error. Unknown keys are rejected with their
// line number.
func Load(path string, explicit bool) (Config, error) {
	c := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return c, c.env()
		}
		return c, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, os.ErrNotExist) && err.Error() != "EOF" {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if err := c.env(); err != nil {
		return c, err
	}
	return c, c.Validate()
}

// env applies `ACCEL_TOKEN` and `ACCEL_LISTEN`, useful in containers.
func (c *Config) env() error {
	if v := os.Getenv("ACCEL_TOKEN"); v != "" {
		c.Token = v
	}
	if v := os.Getenv("ACCEL_LISTEN"); v != "" {
		c.Listen = v
	}
	return nil
}

// Validate rejects values that would misbehave.
func (c Config) Validate() error {
	switch {
	case c.Refresh < 100*time.Millisecond:
		return errors.New("refresh must be at least 100ms")
	case c.History.Resolution <= 0 || c.History.Keep <= 0:
		return errors.New("history.keep and history.resolution must be positive")
	case c.History.Keep/c.History.Resolution > 200000:
		return errors.New("history.keep / history.resolution must stay under 200000 points")
	case c.Thresholds.IdleUtil < 0 || c.Thresholds.IdleUtil > 100 || c.Thresholds.BusyUtil > 100:
		return errors.New("thresholds must be percentages")
	case (c.TLS.Cert == "") != (c.TLS.Key == ""):
		return errors.New("tls.cert and tls.key go together")
	}
	switch c.Alerts.MinSeverity {
	case "", "info", "warning", "critical":
	default:
		return errors.New("alerts.min_severity must be info, warning, or critical")
	}
	for _, n := range c.Nodes {
		if n.Name == "" || (n.URL == "") == (n.SSH == "") {
			return errors.New("every node needs a name and exactly one of url or ssh")
		}
	}
	for _, r := range c.Alerts.Rules {
		if r.When == "" {
			return errors.New("alerts.rules: every rule needs a when clause")
		}
		if len(strings.Fields(r.When)) != 3 {
			return fmt.Errorf("alerts.rules: %q must look like \"util < 10\"", r.When)
		}
	}
	return nil
}

// Loopback reports whether listen binds only to the local machine.
func Loopback(listen string) bool {
	host := listen
	if i := strings.LastIndex(listen, ":"); i >= 0 {
		host = listen[:i]
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}

// Print renders the effective configuration as YAML.
func (c Config) Print() string {
	b, _ := yaml.Marshal(c)
	return string(b)
}
