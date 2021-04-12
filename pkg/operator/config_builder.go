package operator

import (
	"fmt"

	grafana "github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	common "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/pkg/relabel"

	"gopkg.in/yaml.v2"
)

// PrometheusInstance is an instance with a set of associated service monitors,
// pod monitors, and probes, which compose the final configuration of the
// generated Prometheus instance.
type PrometheusInstance struct {
	Instance        *grafana.PrometheusInstance
	ServiceMonitors []*prom.ServiceMonitor
	PodMonitors     []*prom.PodMonitor
	Probes          []*prom.Probe
}

// BuildConfig creates a set of YAML that represents the configuration for the Agent.
func BuildConfig(agent *grafana.GrafanaAgent, proms []PrometheusInstance) (yaml.MapSlice, error) {
	var ms yaml.MapSlice

	// TODO(rfratto): expose log_level and log_format in CRD
	ms = append(ms, yaml.MapItem{
		Key: "server",
		Value: yaml.MapSlice{
			{Key: "http_listen_port", Value: 8080},
			{Key: "log_level", Value: "info"},
		},
	})

	global, err := buildPrometheusGlobal(agent)
	if err != nil {
		return nil, err
	}

	ms = append(ms, yaml.MapItem{
		Key: "prometheus",
		Value: yaml.MapSlice{
			{Key: "wal_directory", Value: "/var/lib/grafana-agent"},
			{Key: "global", Value: global},
			{Key: "configs", Value: nil}, // TODO(rfratto): generate
		},
	})

	return ms, nil
}

func buildPrometheusGlobal(agent *grafana.GrafanaAgent) (yaml.MapSlice, error) {
	var ms yaml.MapSlice

	if v := agent.Spec.Prometheus.ScrapeInterval; v != "" {
		ms = append(ms, yaml.MapItem{Key: "scrape_interval", Value: v})
	}
	if v := agent.Spec.Prometheus.ScrapeTimeout; v != "" {
		ms = append(ms, yaml.MapItem{Key: "scrape_timeout", Value: v})
	}
	if v := agent.Spec.Prometheus.ExternalLabels; len(v) > 0 {
		ms = append(ms, yaml.MapItem{Key: "external_labels", Value: v})
	}
	if v := agent.Spec.Prometheus.RemoteWrite; len(v) > 0 {
		/*
			var items []yaml.MapSlice
			for _, rw := range v {
				rwYAML, err := buildRemoteWrite(&rw)
				if err != nil {
					return nil, err
				}
				items = append(items, rwYAML)
			}
			ms = append(ms, yaml.MapItem{Key: "remote_write", Value: items})
		*/
	}

	return ms, nil
}

func generateRemoteWrite(rw *grafana.RemoteWriteSpec) (*config.RemoteWriteConfig, error) {
	var url *common.URL
	if err := yaml.Unmarshal([]byte(rw.URL), &url); err != nil {
		return nil, fmt.Errorf("couldn't parse remote_write url: %w", err)
	}

	var remoteTimeout model.Duration
	if err := yaml.Unmarshal([]byte(rw.RemoteTimeout), &remoteTimeout); err != nil {
		return nil, fmt.Errorf("couldn't parse remote_timeout: %w", err)
	}

	var relabels []*relabel.Config
	for _, relabel := range rw.WriteRelabelConfigs {
		converted, err := generateRelabelConfig(&relabel)
		if err != nil {
			return nil, err
		}
		relabels = append(relabels, converted)
	}

	var proxyURL common.URL
	if rw.ProxyURL != "" {
		if err := yaml.Unmarshal([]byte(rw.ProxyURL), &proxyURL); err != nil {
			return nil, fmt.Errorf("failed to parse proxy_url: %w", err)
		}
	}

	// TODO(rfratto): SigV4
	return &config.RemoteWriteConfig{
		Name:                rw.Name,
		URL:                 url,
		RemoteTimeout:       remoteTimeout,
		Headers:             rw.Headers,
		WriteRelabelConfigs: relabels,
		HTTPClientConfig: common.HTTPClientConfig{
			BasicAuth: &common.BasicAuth{
				// TODO(rfratto): how does this work? Username is a SecretKeySelector
				// but we can't load usernames from files.
			},
			TLSConfig: common.TLSConfig{
				// TODO(rfratto): also how does this work? same problem as above.
			},
			BearerToken:     common.Secret(rw.BearerToken),
			BearerTokenFile: rw.BearerTokenFile,
			ProxyURL:        proxyURL,
		},
	}, nil
}

func generateRelabelConfig(cfg *grafana.RelabelConfig) (*relabel.Config, error) {
	var sourceLabels model.LabelNames
	for _, lbl := range cfg.SourceLabels {
		sourceLabels = append(sourceLabels, model.LabelName(lbl))
	}

	var regex relabel.Regexp
	if err := yaml.Unmarshal([]byte(cfg.Regex), &regex); err != nil {
		return nil, fmt.Errorf("failed to parse regexp: %w", err)
	}

	var action relabel.Action
	if err := yaml.Unmarshal([]byte(cfg.Action), &action); err != nil {
		return nil, fmt.Errorf("failed to parse relabel_config action: %w", err)
	}

	return &relabel.Config{
		SourceLabels: sourceLabels,
		Separator:    cfg.Separator,
		Regex:        regex,
		Modulus:      cfg.Modulus,
		TargetLabel:  cfg.TargetLabel,
		Replacement:  cfg.Replacement,
		Action:       action,
	}, nil
}
