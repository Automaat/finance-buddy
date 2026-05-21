package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Proxy is the http.Handler that picks an upstream per request.
type Proxy struct {
	rules    []compiledRule
	fallback *httputil.ReverseProxy
	logger   *slog.Logger
}

type compiledRule struct {
	method     string // "*" or upper-case verb
	pathPrefix string
	upstream   *url.URL
	rp         *httputil.ReverseProxy
}

// NewProxy compiles a config into a ready-to-serve handler.
func NewProxy(cfg *Config, logger *slog.Logger) (*Proxy, error) {
	if logger == nil {
		logger = slog.Default()
	}
	def, err := url.Parse(cfg.Default)
	if err != nil {
		return nil, fmt.Errorf("parse default upstream: %w", err)
	}
	rules := make([]compiledRule, 0, len(cfg.Rules))
	for i, r := range cfg.Rules {
		u, err := url.Parse(r.Upstream)
		if err != nil {
			return nil, fmt.Errorf("rule[%d] parse upstream: %w", i, err)
		}
		rules = append(rules, compiledRule{
			method:     strings.ToUpper(r.Method),
			pathPrefix: r.PathPrefix,
			upstream:   u,
			rp:         httputil.NewSingleHostReverseProxy(u),
		})
	}
	return &Proxy{
		rules:    rules,
		fallback: httputil.NewSingleHostReverseProxy(def),
		logger:   logger,
	}, nil
}

// ServeHTTP routes a request to the matching upstream, or the default.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rule, matched := p.match(r)
	if !matched {
		p.fallback.ServeHTTP(w, r)
		return
	}
	p.logger.Debug("routed",
		"method", r.Method,
		"path", r.URL.Path,
		"upstream", rule.upstream.String(),
	)
	rule.rp.ServeHTTP(w, r)
}

func (p *Proxy) match(r *http.Request) (compiledRule, bool) {
	for _, rule := range p.rules {
		if rule.method != "*" && !strings.EqualFold(rule.method, r.Method) {
			continue
		}
		if strings.HasPrefix(r.URL.Path, rule.pathPrefix) {
			return rule, true
		}
	}
	return compiledRule{}, false
}
