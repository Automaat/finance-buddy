package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config controls per-route upstream selection.
//
// Default is the URL anything unmatched falls through to (typically the
// Python backend during cutover). Rules are evaluated in order; the first
// match wins. Each rule specifies an HTTP method (or "*") and a path prefix.
type Config struct {
	Default string `yaml:"default"`
	Rules   []Rule `yaml:"rules"`
}

// Rule describes one cutover target.
//
// Method may be a verb ("GET") or "*" for any method.
// PathPrefix is matched against the request path with strings.HasPrefix —
// trailing slashes matter, so prefer explicit forms like "/api/config".
// Upstream is the target URL (scheme + host + optional path prefix).
type Rule struct {
	Method     string `yaml:"method"`
	PathPrefix string `yaml:"path_prefix"`
	Upstream   string `yaml:"upstream"`
}

// LoadConfig reads and validates a routes.yaml file from disk.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &cfg, nil
}

// Validate checks that the parsed config is internally consistent.
func (c *Config) Validate() error {
	if c.Default == "" {
		return fmt.Errorf("default upstream is required")
	}
	if err := validateUpstream(c.Default); err != nil {
		return fmt.Errorf("default upstream: %w", err)
	}
	for i, r := range c.Rules {
		if r.Method == "" {
			return fmt.Errorf("rule[%d]: method is required (use \"*\" for any)", i)
		}
		if r.Method != "*" && !isHTTPMethod(r.Method) {
			return fmt.Errorf("rule[%d]: %q is not a known HTTP method", i, r.Method)
		}
		if r.PathPrefix == "" || !strings.HasPrefix(r.PathPrefix, "/") {
			return fmt.Errorf("rule[%d]: path_prefix must start with \"/\"", i)
		}
		if r.Upstream == "" {
			return fmt.Errorf("rule[%d]: upstream is required", i)
		}
		if err := validateUpstream(r.Upstream); err != nil {
			return fmt.Errorf("rule[%d] upstream: %w", i, err)
		}
	}
	return nil
}

// validateUpstream rejects anything that isn't an absolute http(s) URL. The
// stdlib url.Parse accepts relative forms like "backend:8000" without error,
// which then trips up httputil.ReverseProxy at runtime — catch it here.
func validateUpstream(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("not a valid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("host is required (e.g. http://backend:8000)")
	}
	return nil
}

func isHTTPMethod(s string) bool {
	switch strings.ToUpper(s) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
