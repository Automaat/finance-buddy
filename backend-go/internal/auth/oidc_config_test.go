package auth

import (
	"errors"
	"maps"
	"strings"
	"testing"
)

func TestLoadOIDCConfig(t *testing.T) {
	full := map[string]string{
		"FB_OIDC_ISSUER_URL":    "https://auth.example.test",
		"FB_OIDC_CLIENT_ID":     "finance",
		"FB_OIDC_CLIENT_SECRET": "s3cret",
		"FB_OIDC_REDIRECT_URL":  "https://finance.example.test/api/auth/oidc/callback",
	}
	without := func(keys ...string) map[string]string {
		env := maps.Clone(full)
		for _, k := range keys {
			delete(env, k)
		}
		return env
	}

	tests := []struct {
		name        string
		env         map[string]string
		wantEnabled bool
		wantMissing []string
	}{
		{name: "all set", env: full, wantEnabled: true},
		{name: "none set", env: map[string]string{}},
		{
			name: "blank values count as unset",
			env: map[string]string{
				"FB_OIDC_ISSUER_URL": " ", "FB_OIDC_CLIENT_ID": "",
				"FB_OIDC_CLIENT_SECRET": "", "FB_OIDC_REDIRECT_URL": "\t",
			},
		},
		{
			name:        "secret missing",
			env:         without("FB_OIDC_CLIENT_SECRET"),
			wantMissing: []string{"FB_OIDC_CLIENT_SECRET"},
		},
		{
			name:        "only issuer set",
			env:         map[string]string{"FB_OIDC_ISSUER_URL": "https://auth.example.test"},
			wantMissing: []string{"FB_OIDC_CLIENT_ID", "FB_OIDC_CLIENT_SECRET", "FB_OIDC_REDIRECT_URL"},
		},
		{
			name:        "issuer missing",
			env:         without("FB_OIDC_ISSUER_URL"),
			wantMissing: []string{"FB_OIDC_ISSUER_URL"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadOIDCConfig(func(k string) string { return tt.env[k] })
			if len(tt.wantMissing) > 0 {
				if !errors.Is(err, ErrOIDCPartialConfig) {
					t.Fatalf("err = %v, want ErrOIDCPartialConfig", err)
				}
				for _, k := range tt.wantMissing {
					if !strings.Contains(err.Error(), k) {
						t.Errorf("error %q does not name %s", err, k)
					}
				}
				if cfg.Enabled() {
					t.Error("partial config must not enable OIDC")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Enabled() != tt.wantEnabled {
				t.Fatalf("Enabled() = %v, want %v", cfg.Enabled(), tt.wantEnabled)
			}
			if tt.wantEnabled && (cfg.IssuerURL != full["FB_OIDC_ISSUER_URL"] ||
				cfg.ClientID != "finance" || cfg.ClientSecret != "s3cret" ||
				cfg.RedirectURL != full["FB_OIDC_REDIRECT_URL"]) {
				t.Fatalf("config mismatch: %+v", cfg)
			}
		})
	}
}
