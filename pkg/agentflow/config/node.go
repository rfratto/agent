package config

import (
	"fmt"
	"time"

	"github.com/grafana/regexp"
	"github.com/prometheus/common/model"
)

type Config struct {
	Nodes []Node `yaml:"nodes,omitempty"`
}

// Node represents a single node, and can be of any type. So the number of configs listed is all of them
// the orchestrator will loop through checking which one it is an initiating it
type Node struct {
	Name string `yaml:"name,omitempty"`
	// Outputs is an array of what references this node, it goes by the unique name
	Outputs []string `yaml:"outputs,omitempty"`

	MetricGenerator *MetricGenerator `yaml:"metric_generator,omitempty"`
	MetricFilter    *MetricFilter    `yaml:"metric_filter,omitempty"`

	AgentLogs     *AgentLogs     `yaml:"agent_logs,omitempty"`
	LogFileWriter *LogFileWriter `yaml:"log_file_writer,omitempty"`

	Github *Github `yaml:"github,omitempty"`

	FakeMetricRemoteWrite *FakeRemoteWrite       `yaml:"fake_metric_remote_write,omitempty"`
	SimpleRemoteWrite     *SimpleRemoteWrite     `yaml:"simple_metric_remote_write,omitempty"`
	PrometheusRemoteWrite *PrometheusRemoteWrite `yaml:"prometheus_remote_write,omitempty"`
	Scraper               *Scraper               `yaml:"scraper,omitempty"`
	MetricRelabelConfigs  *MetricRelabel         `yaml:"metric_relabel_configs,omitempty"`

	Credentials *CredentialsManager `yaml:"credentials,omitempty"`
}

type MetricGenerator struct {
	Format        string        `yaml:"format"`
	SpawnInterval time.Duration `yaml:"spawn_interval,omitempty"`
}

type MetricFilter struct {
	Filters []MetricFilterFilter `yaml:"filters,omitempty"`
}

type MetricFilterFilter struct {
	MatchField string `yaml:"match_field,omitempty"`
	Action     string `yaml:"action,omitempty"`
	Regex      string `yaml:"regex,omitempty"`
	AddValue   string `yaml:"add_value,omitempty"`
	AddLabel   string `yaml:"add_label,omitempty"`
}

type FakeRemoteWrite struct {
	Credential BasicAuthCredential `yaml:"credential,omitempty"`
}

type SimpleRemoteWrite struct {
	URL string `yaml:"url,omitempty"`
}

type CredentialsManager struct {
	// Allow loading of credentials via a file
	File string `yaml:"file,omitempty"`
	// Or allow them to be embedded in the configuration
	Credentials *Credentials `yaml:"credentials,omitempty"`
}

// Credentials is a master credentials object that can be passed between nodes
// Each node will interpret the message and extract the credential that it needs
// for example mysql if given an unnamed credential would use that one, but if there is a named
// one that matches the component name it would use that
type Credentials struct {
	BasicAuth []*BasicAuthCredential `yaml:"basic_auth,omitempty"`
	Redis     []*RedisCredential     `yaml:"redis,omitempty"`
	Github    []*GithubCredential    `yaml:"github,omitempty"`
	MySQL     []*MySQLCredential     `yaml:"mysql,omitempty"`
}

type BasicAuthCredential struct {
	Name     string `yaml:"name,omitempty"`
	URL      string `yaml:"url,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type GithubCredential struct {
	Name     string `yaml:"name,omitempty"`
	APIToken string `yaml:"api_token,omitempty"`
}

type RedisCredential struct {
	Name string `yaml:"name,omitempty"`
	Auth string `yaml:"auth,omitempty"`
}

type MySQLCredential struct {
	Name     string `yaml:"name,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type AgentLogs struct {
}

type LogFileWriter struct {
	Path string `yaml:"path,omitempty"`
}

type Github struct {
	ApiURL         string   `yaml:"api_url,omitempty"`
	Repositories   []string `yaml:"repositories,omitempty"`
	EnableEndpoint bool     `yaml:"enable_endpoint,omitempty"`
}

type PrometheusRemoteWrite struct {
	WalDir   string `yaml:"wal_dir,omitempty"`
	URL      string `yaml:"url,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type FileServiceDiscovery struct {
	Locations       []string       `yaml:"locations,omitempty"`
	RefreshInterval model.Duration `yaml:"refresh_interval,omitempty"`
}

type Scraper struct {
	WalDir        string         `yaml:"wal_dir,omitempty"`
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs,omitempty"`
}

type ScrapeConfig struct {
	JobName        string        `yaml:"job_name,omitempty"`
	ScrapeInterval time.Duration `yaml:"scrape_interval,omitempty"`
	ScrapeTimeout  time.Duration `yaml:"scrape_timeout,omitempty"`
	MetricsPath    string        `yaml:"metrics_path,omitempty"`
	Targets        []string      `yaml:"targets,omitempty"`
}

func (s *ScrapeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*s = ScrapeConfig{
		ScrapeInterval: 60 * time.Second,
		ScrapeTimeout:  10 * time.Second,
		MetricsPath:    "/metrics",
	}
	type plain ScrapeConfig
	return unmarshal((*plain)(s))
}

type MetricRelabel struct {
	Rules []MetricRelabelRule `yaml:"rules,omitempty"`
}

type MetricRelabelRule struct {
	SourceLabels []string `yaml:"source_labels,omitempty"`
	Separator    string   `yaml:"separator,omitempty"`
	TargetLabel  string   `yaml:"target_label,omitempty"`
	Regex        Regex    `yaml:"regex,omitempty"`
	Modulus      uint64   `yaml:"modulus,omitempty"`
	Replacement  string   `yaml:"replacement,omitempty"`
	Action       string   `yaml:"action,omitempty"`
}

var DefaultMetricRelabelRule = MetricRelabelRule{
	Separator:   ";",
	Regex:       Regex{regexp.MustCompile("(.*)")},
	Replacement: "$1",
	Action:      "replace",
}

func (rr *MetricRelabelRule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*rr = DefaultMetricRelabelRule

	type rule MetricRelabelRule
	if err := unmarshal((*rule)(rr)); err != nil {
		return err
	}

	switch rr.Action {
	case "replace", "keep", "drop", "hashmod", "labelmap", "labeldrop", "labelkeep":
		// no-op
	case "":
		return fmt.Errorf("action is required")
	default:
		return fmt.Errorf("unrecognized action %q", rr.Action)
	}

	if rr.Action == "replace" && rr.TargetLabel == "" {
		return fmt.Errorf("target_label must be set when action is replace")
	}

	return nil
}

type Regex struct{ *regexp.Regexp }

func (r Regex) String() string {
	if r.Regexp == nil {
		return ""
	}
	return r.Regexp.String()
}

func (r Regex) MarshalYAML() (interface{}, error) {
	return r.String(), nil
}

func (r *Regex) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	compiled, err := regexp.Compile(s)
	if err != nil {
		return err
	}

	r.Regexp = compiled
	return nil
}
