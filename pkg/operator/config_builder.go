package operator

import (
	"fmt"

	grafana "github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
	"github.com/grafana/agent/pkg/operator/assets"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	common "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/pkg/relabel"

	"gopkg.in/yaml.v2"
)

// TODO(rfratto):
//
// This is a pretty good start, but generating specifically the Prometheus
// configuration types and then marshaling to YAML isn't saving us as much
// work as it initially seemed like it would.
//
// Let's backpedal and go back to generating yaml.MapSlices.

// PrometheusInstance is an instance with a set of associated service monitors,
// pod monitors, and probes, which compose the final configuration of the
// generated Prometheus instance.
type PrometheusInstance struct {
	Instance        *grafana.PrometheusInstance
	ServiceMonitors []*prom.ServiceMonitor
	PodMonitors     []*prom.PodMonitor
	Probes          []*prom.Probe
}

// Deployment is a set of resources used for one deployment of the Agent.
type Deployment struct {
	// Agent is the root resource that the deployment represents.
	Agent *grafana.GrafanaAgent
	// Prometheis is the set of prometheus instances discovered from the root Agent resource.
	Prometheis []PrometheusInstance
}

// BuildConfig builds an Agent configuration file. The secrets store is used
// for secret lookup for any configurations that cannot be directly loaded from a file.
// This includes SigV4 and basic_auth credentials.
//
// If a secret is used but is not found in the store, BuildConfig will fail.
func (d *Deployment) BuildConfig(secrets assets.SecretStore) (yaml.MapSlice, error) {
	if d.Agent == nil {
		return nil, fmt.Errorf("agent must not be nil")
	}

	var ms yaml.MapSlice

	// TODO(rfratto): expose log_level and log_format in CRD
	ms = append(ms, yaml.MapItem{
		Key: "server",
		Value: yaml.MapSlice{
			{Key: "http_listen_port", Value: 8080},
			{Key: "log_level", Value: "info"},
		},
	})

	global, err := buildPrometheusGlobal(d.Agent, secrets)
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

func buildPrometheusGlobal(agent *grafana.GrafanaAgent, secrets assets.SecretStore) (yaml.MapSlice, error) {
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

func generateRemoteWrite(ns string, rw *grafana.RemoteWriteSpec, secrets assets.SecretStore) (*config.RemoteWriteConfig, error) {
	var (
		sendURL  *common.URL
		proxyURL common.URL

		remoteTimeout, batchSendDeadline model.Duration
		minBackoff, maxBackoff           model.Duration
		metadataSendInterval             model.Duration
	)

	parseTargets := []struct {
		name   string
		input  func() []byte
		target interface{}
	}{
		{
			name:   "remote_write url",
			input:  func() []byte { return []byte(rw.URL) },
			target: &sendURL,
		},
		{
			name:   "remote_timeout",
			input:  func() []byte { return []byte(rw.RemoteTimeout) },
			target: &remoteTimeout,
		},
		{
			name:   "proxy_url",
			input:  func() []byte { return []byte(rw.ProxyURL) },
			target: &proxyURL,
		},
		{
			name: "batch_send_deadline",
			input: func() []byte {
				if rw.QueueConfig == nil {
					return nil
				}
				return []byte(rw.QueueConfig.BatchSendDeadline)
			},
			target: &batchSendDeadline,
		},
		{
			name: "min_backoff",
			input: func() []byte {
				if rw.QueueConfig == nil {
					return nil
				}
				return []byte(rw.QueueConfig.MinBackoff)
			},
			target: &minBackoff,
		},
		{
			name: "max_backoff",
			input: func() []byte {
				if rw.QueueConfig == nil {
					return nil
				}
				return []byte(rw.QueueConfig.MaxBackoff)
			},
			target: &maxBackoff,
		},
		{
			name: "metadata send_interval",
			input: func() []byte {
				if rw.MetadataConfig == nil {
					return nil
				}
				return []byte(rw.MetadataConfig.SendInterval)
			},
			target: &metadataSendInterval,
		},
	}
	for _, tgt := range parseTargets {
		bb := tgt.input()
		if len(bb) == 0 {
			continue
		}
		if err := yaml.Unmarshal(bb, tgt.target); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", tgt.name, err)
		}
	}

	var relabels []*relabel.Config
	for _, relabel := range rw.WriteRelabelConfigs {
		converted, err := generateRelabelConfig(&relabel)
		if err != nil {
			return nil, err
		}
		relabels = append(relabels, converted)
	}

	var sigV4 *config.SigV4Config
	if rw.SigV4 != nil {
		sigV4 = &config.SigV4Config{
			Region:  rw.SigV4.Region,
			Profile: rw.SigV4.Profile,
			RoleARN: rw.SigV4.RoleARN,
		}

		if rw.SigV4.AccessKey != nil {
			key := assets.KeyForSecret(ns, rw.SigV4.AccessKey)
			secret, ok := secrets[key]
			if !ok {
				return nil, fmt.Errorf("secret not loaded: %s", key)
			}
			sigV4.AccessKey = secret
		}

		if rw.SigV4.SecretKey != nil {
			key := assets.KeyForSecret(ns, rw.SigV4.SecretKey)
			secret, ok := secrets[key]
			if !ok {
				return nil, fmt.Errorf("secret not loaded: %s", key)
			}
			sigV4.SecretKey = common.Secret(secret)
		}
	}

	return &config.RemoteWriteConfig{
		Name:                rw.Name,
		URL:                 sendURL,
		RemoteTimeout:       remoteTimeout,
		Headers:             rw.Headers,
		WriteRelabelConfigs: relabels,
		HTTPClientConfig: common.HTTPClientConfig{
			BasicAuth: &common.BasicAuth{
				// TODO(rfratto): load in secerts
			},
			TLSConfig: common.TLSConfig{
				CAFile:             rw.TLSConfig.CAFile,
				CertFile:           rw.TLSConfig.CertFile,
				KeyFile:            rw.TLSConfig.KeyFile,
				ServerName:         rw.TLSConfig.ServerName,
				InsecureSkipVerify: rw.TLSConfig.InsecureSkipVerify,
			},
			BearerToken:     common.Secret(rw.BearerToken),
			BearerTokenFile: rw.BearerTokenFile,
			ProxyURL:        proxyURL,
		},
		SigV4Config: sigV4,
		QueueConfig: config.QueueConfig{
			Capacity:          rw.QueueConfig.Capacity,
			MaxShards:         rw.QueueConfig.MaxShards,
			MinShards:         rw.QueueConfig.MinShards,
			MaxSamplesPerSend: rw.QueueConfig.MaxSamplesPerSend,
			BatchSendDeadline: batchSendDeadline,
			MinBackoff:        minBackoff,
			MaxBackoff:        maxBackoff,
			RetryOnRateLimit:  rw.QueueConfig.RetryOnRateLimit,
		},
		MetadataConfig: config.MetadataConfig{
			Send:         rw.MetadataConfig.Send,
			SendInterval: metadataSendInterval,
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
