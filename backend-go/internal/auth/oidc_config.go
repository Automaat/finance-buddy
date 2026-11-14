package auth

import (
	"errors"
	"fmt"
	"strings"
)

// ErrOIDCPartialConfig is returned when only some FB_OIDC_* variables are set.
var ErrOIDCPartialConfig = errors.New("incomplete OIDC configuration")

// OIDCConfig is the optional OpenID Connect (Authelia) single sign-on setup.
// The zero value means OIDC login is disabled.
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Enabled reports whether OIDC login is configured.
func (c OIDCConfig) Enabled() bool {
	return c.IssuerURL != ""
}

var oidcEnvVars = [...]string{
	"FB_OIDC_ISSUER_URL",
	"FB_OIDC_CLIENT_ID",
	"FB_OIDC_CLIENT_SECRET",
	"FB_OIDC_REDIRECT_URL",
}

// LoadOIDCConfig reads the FB_OIDC_* variables through getenv. All four set
// enables OIDC login, none set leaves it disabled, and a partial set is an
// error naming the missing variables.
func LoadOIDCConfig(getenv func(string) string) (OIDCConfig, error) {
	var vals [len(oidcEnvVars)]string
	var missing []string
	for i, key := range oidcEnvVars {
		vals[i] = strings.TrimSpace(getenv(key))
		if vals[i] == "" {
			missing = append(missing, key)
		}
	}
	switch len(missing) {
	case 0:
		return OIDCConfig{
			IssuerURL:    vals[0],
			ClientID:     vals[1],
			ClientSecret: vals[2],
			RedirectURL:  vals[3],
		}, nil
	case len(oidcEnvVars):
		return OIDCConfig{}, nil
	default:
		return OIDCConfig{}, fmt.Errorf("%w: set all FB_OIDC_* variables or none; missing %s",
			ErrOIDCPartialConfig, strings.Join(missing, ", "))
	}
}
