// Package http contains configuration for components which manage HTTP
// clients. It is an extension over [github.com/prometheus/common/config].
package http

import (
	"encoding"
	"fmt"
	"net/url"
	"strings"

	"github.com/grafana/agent/pkg/flow/rivertypes"
	"github.com/grafana/agent/pkg/river"
	"github.com/prometheus/common/config"
)

const bearerAuth string = "Bearer"

// ClientConfig configures an HTTP client.
type ClientConfig struct {
	BasicAuth       *BasicAuth        `river:"basic_auth,block,optional"`
	Authorization   *Authorization    `river:"authorization,block,optional"`
	OAuth2          *OAuth2           `river:"oauth2,block,optional"`
	BearerToken     rivertypes.Secret `river:"bearer_token,attr,optional"`
	BearerTokenFile string            `river:"bearer_token_file,attr,optional"`
	ProxyURL        URL               `river:"proxy_url,attr,optional"`
	TLSConfig       TLSConfig         `river:"tls_config,block,optional"`
	FollowRedirects bool              `river:"follow_redirects,attr,optional"`
	EnableHTTP2     bool              `river:"enable_http2,attr,optional"`
}

// DefaultClientConfig create a default client configuration.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		FollowRedirects: true,
		EnableHTTP2:     true,
	}
}

var _ river.Unmarshaler = (*ClientConfig)(nil)

// UnmarshalRiver applies defaults to the ClientConfig and validates it.
func (cc *ClientConfig) UnmarshalRiver(f func(v interface{}) error) error {
	*cc = *DefaultClientConfig()
	type config ClientConfig
	if err := f((*config)(cc)); err != nil {
		return err
	}

	return cc.Validate()
}

// Validate returns an error if the ClientConfig is invalid.
func (cc *ClientConfig) Validate() error {
	// Backwards compatibility with the bearer_token field.
	if len(cc.BearerToken) > 0 && len(cc.BearerTokenFile) > 0 {
		return fmt.Errorf("at most one of bearer_token & bearer_token_file must be configured")
	}
	if (cc.BasicAuth != nil || cc.OAuth2 != nil) && (len(cc.BearerToken) > 0 || len(cc.BearerTokenFile) > 0) {
		return fmt.Errorf("at most one of basic_auth, oauth2, bearer_token & bearer_token_file must be configured")
	}
	if cc.BasicAuth != nil && (string(cc.BasicAuth.Password) != "" && cc.BasicAuth.PasswordFile != "") {
		return fmt.Errorf("at most one of basic_auth password & password_file must be configured")
	}
	if cc.Authorization != nil {
		if len(cc.BearerToken) > 0 || len(cc.BearerTokenFile) > 0 {
			return fmt.Errorf("authorization is not compatible with bearer_token & bearer_token_file")
		}
		if string(cc.Authorization.Credentials) != "" && cc.Authorization.CredentialsFile != "" {
			return fmt.Errorf("at most one of authorization credentials & credentials_file must be configured")
		}
		cc.Authorization.Type = strings.TrimSpace(cc.Authorization.Type)
		if len(cc.Authorization.Type) == 0 {
			cc.Authorization.Type = bearerAuth
		}
		if strings.ToLower(cc.Authorization.Type) == "basic" {
			return fmt.Errorf(`authorization type cannot be set to "basic", use "basic_auth" instead`)
		}
		if cc.BasicAuth != nil || cc.OAuth2 != nil {
			return fmt.Errorf("at most one of basic_auth, oauth2 & authorization must be configured")
		}
	} else {
		if len(cc.BearerToken) > 0 {
			cc.Authorization = &Authorization{Credentials: cc.BearerToken}
			cc.Authorization.Type = bearerAuth
			cc.BearerToken = ""
		}
		if len(cc.BearerTokenFile) > 0 {
			cc.Authorization = &Authorization{CredentialsFile: cc.BearerTokenFile}
			cc.Authorization.Type = bearerAuth
			cc.BearerTokenFile = ""
		}
	}
	if cc.OAuth2 != nil {
		if cc.BasicAuth != nil {
			return fmt.Errorf("at most one of basic_auth, oauth2 & authorization must be configured")
		}
		if len(cc.OAuth2.ClientID) == 0 {
			return fmt.Errorf("oauth2 client_id must be configured")
		}
		if len(cc.OAuth2.ClientSecret) == 0 && len(cc.OAuth2.ClientSecretFile) == 0 {
			return fmt.Errorf("either oauth2 client_secret or client_secret_file must be configured")
		}
		if len(cc.OAuth2.TokenURL) == 0 {
			return fmt.Errorf("oauth2 token_url must be configured")
		}
		if len(cc.OAuth2.ClientSecret) > 0 && len(cc.OAuth2.ClientSecretFile) > 0 {
			return fmt.Errorf("at most one of oauth2 client_secret & client_secret_file must be configured")
		}
	}
	return nil
}

// BasicAuth configures Basic HTTP authentication credentials.
type BasicAuth struct {
	Username     string            `river:"username,attr,optional"`
	Password     rivertypes.Secret `river:"password,attr,optional"`
	PasswordFile string            `river:"password_file,attr,optional"`
}

// URL implements encoding.TextMarshaler and encoding.TextUnmarshaler for a
// URL.
type URL struct {
	*url.URL
}

var (
	_ encoding.TextMarshaler   = URL{}
	_ encoding.TextUnmarshaler = (*URL)(nil)
)

// MarshalText implements encoding.TextMarshaler.
func (u URL) MarshalText() (text []byte, err error) {
	u2 := &config.URL{
		URL: u.URL,
	}
	if u.URL != nil {
		return []byte(u2.Redacted()), nil
	}
	return nil, nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *URL) UnmarshalText(text []byte) error {
	s := string(text)
	urlp, err := url.Parse(s)
	if err != nil {
		return err
	}
	u.URL = urlp
	return nil
}

// Authorization specifies generic HTTP authorization.
type Authorization struct {
	Type            string            `river:"type,attr,optional"`
	Credentials     rivertypes.Secret `river:"credentials,attr,optional"`
	CredentialsFile string            `river:"credentials_file,attr,optional"`
}

// TLSVersion specifies a TLS version to use.
type TLSVersion uint16

// MarshalText implements encoding.TextMarshaler
func (tv TLSVersion) MarshalText() (text []byte, err error) {
	for s, v := range config.TLSVersions {
		if config.TLSVersion(tv) == v {
			return []byte(s), nil
		}
	}
	return nil, fmt.Errorf("unknown TLS version: %d", tv)
}

// UnmarshalText implements encoding.TextUnmarshaler
func (tv *TLSVersion) UnmarshalText(text []byte) error {
	if v, ok := config.TLSVersions[string(text)]; ok {
		*tv = TLSVersion(v)
		return nil
	}
	return fmt.Errorf("unknown TLS version: %s", string(text))
}

// TLSConfig configures options for TLS connections.
type TLSConfig struct {
	CA   rivertypes.OptionalSecret `river:"ca,attr,optional"`
	Cert rivertypes.OptionalSecret `river:"cert,attr,optional"`
	Key  rivertypes.Secret         `river:"key,attr,optional"`

	CAFile   string `river:"ca_file,attr,optional"`
	CertFile string `river:"cert_file,attr,optional"`
	KeyFile  string `river:"key_file,attr,optional"`

	ServerName         string     `river:"server_name,attr,optional"`
	InsecureSkipVerify bool       `river:"insecure_skip_verify,attr,optional"`
	MinVersion         TLSVersion `river:"min_version,attr,optional"`
	MaxVersion         TLSVersion `river:"max_version,attr,optional"`
}

// UnmarshalRiver performs validations at unmarshal time.
func (config *TLSConfig) UnmarshalRiver(f func(interface{}) error) error {
	type tlsConfig TLSConfig
	if err := f((*tlsConfig)(config)); err != nil {
		return err
	}

	return config.Validate()
}

// Validate returns an error if the TLSConfig is invalid.
func (config *TLSConfig) Validate() error {
	switch {
	case config.CA.Value != "" && config.CAFile != "":
		return fmt.Errorf("at most one of ca and ca_file may be set")
	case config.Cert.Value != "" && config.CertFile != "":
		return fmt.Errorf("at most one of cert and cert_file may be set")
	case config.Key != "" && config.KeyFile != "":
		return fmt.Errorf("at most one of key and key_file may be set")
	}

	return nil
}

// OAuth2 configures OAuth2 authentication.
type OAuth2 struct {
	ClientID         string            `river:"client_id,attr,optional"`
	ClientSecret     rivertypes.Secret `river:"client_secret,attr,optional"`
	ClientSecretFile string            `river:"client_secret_file,attr,optional"`
	Scopes           []string          `river:"scopes,attr,optional"`
	TokenURL         string            `river:"token_url,attr,optional"`
	EndpointParams   map[string]string `river:"endpoint_params,attr,optional"`
	ProxyURL         URL               `river:"proxy_url,attr,optional"`
	TLSConfig        TLSConfig         `river:"tls_config,block,optional"`
}
