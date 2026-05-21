package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestProxy(t *testing.T, cfg *Config) *Proxy {
	t.Helper()
	p, err := NewProxy(cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	return p
}

func echoServer(name string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", name)
		_, _ = io.WriteString(w, name+" "+r.Method+" "+r.URL.Path)
	}))
}

func TestProxyFallbackToDefault(t *testing.T) {
	py := echoServer("python")
	defer py.Close()

	cfg := &Config{Default: py.URL}
	srv := httptest.NewServer(newTestProxy(t, cfg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/anything")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("X-Backend"); got != "python" {
		t.Fatalf("unrouted request should hit default; X-Backend = %q", got)
	}
}

func TestProxyRouteToMatchingUpstream(t *testing.T) {
	py := echoServer("python")
	defer py.Close()
	goSrv := echoServer("go")
	defer goSrv.Close()

	cfg := &Config{
		Default: py.URL,
		Rules: []Rule{
			{Method: "GET", PathPrefix: "/api/config", Upstream: goSrv.URL},
		},
	}
	srv := httptest.NewServer(newTestProxy(t, cfg))
	defer srv.Close()

	cases := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		{"matched GET → go", http.MethodGet, "/api/config", "go"},
		{"unmatched method falls back", http.MethodPost, "/api/config", "python"},
		{"unmatched path falls back", http.MethodGet, "/api/accounts", "python"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, srv.URL+tc.path, http.NoBody)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			defer resp.Body.Close()
			if got := resp.Header.Get("X-Backend"); got != tc.expected {
				t.Fatalf("X-Backend = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestProxyWildcardMethod(t *testing.T) {
	py := echoServer("python")
	defer py.Close()
	goSrv := echoServer("go")
	defer goSrv.Close()

	cfg := &Config{
		Default: py.URL,
		Rules: []Rule{
			{Method: "*", PathPrefix: "/api/snapshots", Upstream: goSrv.URL},
		},
	}
	srv := httptest.NewServer(newTestProxy(t, cfg))
	defer srv.Close()

	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		req, _ := http.NewRequest(m, srv.URL+"/api/snapshots/42", http.NoBody)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do %s: %v", m, err)
		}
		_ = resp.Body.Close()
		if got := resp.Header.Get("X-Backend"); got != "go" {
			t.Fatalf("%s X-Backend = %q, want go", m, got)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing default",
			cfg:     Config{},
			wantErr: "default upstream is required",
		},
		{
			name: "rule missing method",
			cfg: Config{
				Default: "http://x",
				Rules:   []Rule{{PathPrefix: "/x", Upstream: "http://y"}},
			},
			wantErr: "method is required",
		},
		{
			name: "rule unknown method",
			cfg: Config{
				Default: "http://x",
				Rules:   []Rule{{Method: "BOGUS", PathPrefix: "/x", Upstream: "http://y"}},
			},
			wantErr: "not a known HTTP method",
		},
		{
			name: "rule path without leading slash",
			cfg: Config{
				Default: "http://x",
				Rules:   []Rule{{Method: "GET", PathPrefix: "api/x", Upstream: "http://y"}},
			},
			wantErr: "path_prefix must start with",
		},
		{
			name: "rule missing upstream",
			cfg: Config{
				Default: "http://x",
				Rules:   []Rule{{Method: "GET", PathPrefix: "/x"}},
			},
			wantErr: "upstream is required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err, tc.wantErr)
			}
		})
	}
}
