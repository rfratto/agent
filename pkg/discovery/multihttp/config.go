package multihttp

import (
	"fmt"
	"net/url"
	"time"

	common_config "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/pkg/relabel"
)

// APIPathLabel is the name of the label that holds the path where a service
// exposes its http_sd API.
const APIPathLabel = "__api_path__"

// DefaultConfig for multihttp discovery.
var DefaultConfig = Config{
	RefreshInterval:  model.Duration(60 * time.Second),
	RefreshTimeout:   model.Duration(10 * time.Second),
	APIPath:          "/targets",
	Scheme:           "http",
	HTTPClientConfig: common_config.DefaultHTTPClientConfig,
}

// Config configures multihttp discovery.
type Config struct {
	HonorLabels     bool           `yaml:"honor_labels,omitempty"`     // Leave labels from targets unmodified
	Params          url.Values     `yaml:"params,omitempty"`           // Extra query parameters to pass to service API
	RefreshInterval model.Duration `yaml:"refresh_interval,omitempty"` // Frequency to poll for new targets from service
	RefreshTimeout  model.Duration `yaml:"refresh_timeout,omitempty"`  // Timeout for polling for new targets from service
	APIPath         string         `yaml:"api_path,omitempty"`         // HTTP resource path where targets are exposed from a service
	Scheme          string         `yaml:"scheme,omitempty"`           // URL scheme to use for fetching targets from services

	ServiceDiscoveryConfigs discovery.Configs              `yaml:"-"`
	HTTPClientConfig        common_config.HTTPClientConfig `yaml:",inline"`

	// Relabel rules for discovered http_sd services.
	RelabelConfigs []*relabel.Config `yaml:"relabel_configs,omitempty"`
	// Relabel rules for targets returned by http_sd services.
	TargetRelabelConfigs []*relabel.Config `yaml:"target_relabel_configs,omitempty"`
}

var _ discovery.Config = (*Config)(nil)

// Name implements discovery.Config.
func (c *Config) Name() string { return "multihttp" }

// NewDiscoverer implements discovery.Config.
func (c *Config) NewDiscoverer(o discovery.DiscovererOptions) (discovery.Discoverer, error) {
	return NewDiscovery(o.Logger, c)
}

// UnmarshalYAML implements the yaml.Unmarshaler interface and validates the
// unmarshaled config.
func (c *Config) UmmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultConfig

	if err := discovery.UnmarshalYAMLWithInlineConfigs(c, unmarshal); err != nil {
		return err
	}

	// The UnmarshalYAML method of HTTPClientConfig is not being called because
	// it's not a pointer. We cannot make it a pointer as the parser panics for
	// inlined pointer structs. Thus we just do its validation here.
	if err := c.HTTPClientConfig.Validate(); err != nil {
		return err
	}

	for i, rc := range c.RelabelConfigs {
		if rc == nil {
			return fmt.Errorf("empty or null service relabeling rule in multihttp config at index %d", i)
		}
	}
	for i, rc := range c.TargetRelabelConfigs {
		if rc == nil {
			return fmt.Errorf("empty or null target relabeling rule in multihttp config at index %d", i)
		}
	}

	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (c *Config) MarshalYAML() (interface{}, error) {
	return discovery.MarshalYAMLWithInlineConfigs(c)
}
