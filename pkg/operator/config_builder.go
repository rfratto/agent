package operator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	grafana "github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
	"github.com/grafana/agent/pkg/operator/assets"
	"github.com/grafana/agent/pkg/util"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
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
func (d *Deployment) BuildConfig(secrets assets.SecretStore) (ms util.Map, err error) {
	if d.Agent == nil {
		return nil, fmt.Errorf("agent must not be nil")
	}

	{
		var server util.Map
		server.Set("http_listen_port", 8080)

		if d.Agent.Spec.LogLevel != "" {
			server.Set("log_level", d.Agent.Spec.LogLevel)
		}
		if d.Agent.Spec.LogFormat != "" {
			server.Set("log_format", d.Agent.Spec.LogFormat)
		}

		ms.Set("server", server)
	}

	global, err := buildPrometheusGlobal(d.Agent.Namespace, &d.Agent.Spec.Prometheus, secrets)
	if err != nil {
		return nil, fmt.Errorf("failed generating prometheus global config: %w", err)
	}

	shards := int32(1)
	if numShards := d.Agent.Spec.Prometheus.Shards; numShards != nil && *numShards > 1 {
		shards = *numShards
	}

	var instances []util.Map
	for _, instance := range d.Prometheis {
		inst, err := generatePrometheusInstance(
			d.Agent.Namespace,
			&instance,
			d.Agent.Spec.APIServerConfig,
			d.Agent.Spec.Prometheus.OverrideHonorLabels,
			d.Agent.Spec.Prometheus.OverrideHonorTimestamps,
			d.Agent.Spec.Prometheus.IgnoreNamespaceSelectors,
			d.Agent.Spec.Prometheus.EnforcedNamepsaceLabel,
			d.Agent.Spec.Prometheus.EnforcedSampleLimit,
			d.Agent.Spec.Prometheus.EnforcedTargetLimit,
			shards,
			secrets,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"error generating prometheus instance %s/%s: %w",
				instance.Instance.Namespace,
				instance.Instance.Name,
				err,
			)
		}
		instances = append(instances, inst)
	}

	ms.Set("prometheus", util.Map{
		{Key: "wal_directory", Value: "/var/lib/grafana-agent/data"},
		{Key: "global", Value: global},
		{Key: "configs", Value: instances},
	})

	return
}

func buildPrometheusGlobal(ns string, cfg *grafana.PrometheusSubsystemSpec, secrets assets.SecretStore) (ms util.Map, err error) {
	ms.Set("external_labels", cfg.ExternalLabels)

	if cfg.ScrapeInterval != "" {
		ms.Set("scrape_interval", cfg.ScrapeInterval)
	}
	if cfg.ScrapeTimeout != "" {
		ms.Set("scrape_timeout", cfg.ScrapeTimeout)
	}
	if len(cfg.RemoteWrite) > 0 {
		var remoteWrites []util.Map

		for i, rw := range cfg.RemoteWrite {
			rwYAML, err := generateRemoteWrite(ns, &rw, secrets)
			if err != nil {
				return nil, fmt.Errorf("error generating remote_write %d: %w", i, err)
			}
			remoteWrites = append(remoteWrites, rwYAML)
		}

		ms.Set("remote_write", remoteWrites)
	}

	return
}

func generateRemoteWrite(ns string, rw *grafana.RemoteWriteSpec, secrets assets.SecretStore) (ms util.Map, err error) {
	ms.Set("url", rw.URL)
	if rw.RemoteTimeout != "" {
		ms.Set("remote_timeout", rw.RemoteTimeout)
	}
	if len(rw.Headers) > 0 {
		ms.Set("headers", rw.Headers)
	}
	if len(rw.WriteRelabelConfigs) > 0 {
		var configs []util.Map
		for _, cfg := range rw.WriteRelabelConfigs {
			configs = append(configs, generateRelabelConfig(&cfg))
		}
		ms.Set("write_relabel_configs", configs)
	}
	if rw.Name != "" {
		ms.Set("name", rw.Name)
	}
	if rw.BasicAuth != nil {
		username, ok := secrets[assets.KeyForSecret(ns, &rw.BasicAuth.Username)]
		if !ok {
			return nil, fmt.Errorf("expected basicAuth username secret to be provided")
		}
		password, ok := secrets[assets.KeyForSecret(ns, &rw.BasicAuth.Password)]
		if !ok {
			return nil, fmt.Errorf("expected basicAuth password secret to be provided")
		}

		ms.Set("basic_auth", util.Map{
			{Key: "username", Value: username},
			{Key: "password", Value: password},
		})
	}
	if rw.TLSConfig != nil {
		ms.Set("tls_config", generateTLSConfig(ns, rw.TLSConfig))
	}
	// TODO(rfratto): follow_redirects
	if rw.QueueConfig != nil {
		var queueConfig util.Map

		if rw.QueueConfig.Capacity != 0 {
			queueConfig.Set("capacity", rw.QueueConfig.Capacity)
		}
		if rw.QueueConfig.MaxShards != 0 {
			queueConfig.Set("max_shards", rw.QueueConfig.MaxShards)
		}
		if rw.QueueConfig.MinShards != 0 {
			queueConfig.Set("min_shards", rw.QueueConfig.MinShards)
		}
		if rw.QueueConfig.MaxSamplesPerSend != 0 {
			queueConfig.Set("max_samples_per_send", rw.QueueConfig.MaxSamplesPerSend)
		}
		if rw.QueueConfig.BatchSendDeadline != "" {
			queueConfig.Set("batch_send_deadline", rw.QueueConfig.BatchSendDeadline)
		}
		if rw.QueueConfig.MinBackoff != "" {
			queueConfig.Set("min_backoff", rw.QueueConfig.MinBackoff)
		}
		if rw.QueueConfig.MaxBackoff != "" {
			queueConfig.Set("max_backoff", rw.QueueConfig.MaxBackoff)
		}
		// TODO(rfratto): retry_on_http_429, which is currently experimental.

		ms.Set("queue_config", queueConfig)
	}
	if rw.MetadataConfig != nil {
		var metadataConfig util.Map
		metadataConfig.Set("send", rw.MetadataConfig.Send)

		if rw.MetadataConfig.SendInterval != "" {
			metadataConfig.Set("send_interval", rw.MetadataConfig.SendInterval)
		}

		ms.Set("metadata_config", metadataConfig)
	}
	if rw.SigV4 != nil {
		var sigV4 util.Map

		if rw.SigV4.Region != "" {
			sigV4.Set("region", rw.SigV4.Region)
		}
		if rw.SigV4.AccessKey != nil {
			val, ok := secrets[assets.KeyForSecret(ns, rw.SigV4.AccessKey)]
			if !ok {
				return nil, fmt.Errorf("couldn't find sigv4 accessKey")
			}
			sigV4.Set("access_key", val)
		}
		if rw.SigV4.SecretKey != nil {
			val, ok := secrets[assets.KeyForSecret(ns, rw.SigV4.SecretKey)]
			if !ok {
				return nil, fmt.Errorf("couldn't find sigv4 secretKey")
			}
			sigV4.Set("secret_key", val)
		}
		if rw.SigV4.Profile != "" {
			sigV4.Set("profile", rw.SigV4.Profile)
		}
		if rw.SigV4.RoleARN != "" {
			sigV4.Set("role_arn", rw.SigV4.RoleARN)
		}

		ms.Set("sigv4", sigV4)
	}

	return
}

func generateRelabelConfig(cfg *prom.RelabelConfig) (ms util.Map) {
	ms.Set("source_labels", cfg.SourceLabels)
	if cfg.Separator != "" {
		ms.Set("separator", cfg.Separator)
	}
	if cfg.Regex != "" {
		ms.Set("regex", cfg.Regex)
	}
	if cfg.Modulus != 0 {
		ms.Set("modulus", cfg.Modulus)
	}
	if cfg.TargetLabel != "" {
		ms.Set("target_label", cfg.TargetLabel)
	}
	if cfg.Replacement != "" {
		ms.Set("replacement", cfg.Replacement)
	}
	if cfg.Action != "" {
		ms.Set("action", cfg.Action)
	}
	return
}

func generateTLSConfig(ns string, cfg *prom.TLSConfig) (ms util.Map) {
	ms = append(ms, generateSafeTLSConfig(ns, &cfg.SafeTLSConfig)...)
	if cfg.CAFile != "" {
		ms.Set("ca_file", cfg.CAFile)
	}
	if cfg.CertFile != "" {
		ms.Set("cert_file", cfg.CertFile)
	}
	if cfg.KeyFile != "" {
		ms.Set("key_file", cfg.KeyFile)
	}
	return
}

func generateSafeTLSConfig(ns string, cfg *prom.SafeTLSConfig) (ms util.Map) {
	if cfg.CA.Secret != nil || cfg.CA.ConfigMap != nil {
		ms.Set("ca_file", pathForKey(assets.KeyForSelector(ns, &cfg.CA)))
	}
	if cfg.Cert.Secret != nil || cfg.Cert.ConfigMap != nil {
		ms.Set("cert_file", pathForKey(assets.KeyForSelector(ns, &cfg.Cert)))
	}
	if cfg.KeySecret != nil {
		ms.Set("key_file", pathForKey(assets.KeyForSecret(ns, cfg.KeySecret)))
	}
	if cfg.ServerName != "" {
		ms.Set("sever_name", cfg.ServerName)
	}
	ms.Set("insecure_skip_verify", cfg.InsecureSkipVerify)
	return
}

func pathForKey(key assets.Key) string {
	return filepath.Join("/var/lib/grafana-agent/secrets", SanitizeLabelName(string(key)))
}

func generatePrometheusInstance(
	agentNS string,
	cfg *PrometheusInstance,
	apiServer *prom.APIServerConfig,
	overrideHonorLabels bool,
	overrideHonorTimestamps bool,
	ignoreNamespaceSelectors bool,
	enforcedNamespaceLabel string,
	enforcedSampleLimit *uint64,
	enforcedTargetLimit *uint64,
	shards int32,
	secrets assets.SecretStore,
) (ms util.Map, err error) {

	// Some abbreviated variable names for sanity
	var (
		ns   = cfg.Instance.Namespace
		spec = cfg.Instance.Spec
	)

	ms.Set("name", fmt.Sprintf("%s/%s", ns, cfg.Instance.Name))

	if spec.WALTruncateFrequency != "" {
		ms.Set("wal_truncate_frequency", spec.WALTruncateFrequency)
	}
	if spec.MinWALTime != "" {
		ms.Set("min_wal_time", spec.MinWALTime)
	}
	if spec.MaxWALTime != "" {
		ms.Set("max_wal_time", spec.MaxWALTime)
	}
	if spec.RemoteFlushDeadline != "" {
		ms.Set("remote_flush_deadline", spec.RemoteFlushDeadline)
	}
	if spec.WriteStaleOnShutdown != nil {
		ms.Set("write_stale_on_shutdown", *spec.WriteStaleOnShutdown)
	}

	var scrapeConfigs []util.Map

	// Generate scrape_configs for everything discovered
	for _, sMon := range cfg.ServiceMonitors {
		for i, ep := range sMon.Spec.Endpoints {
			gen, err := generateServiceMonitor(
				agentNS,
				sMon,
				&ep,
				i,
				apiServer,
				overrideHonorLabels,
				overrideHonorTimestamps,
				ignoreNamespaceSelectors,
				enforcedNamespaceLabel,
				enforcedSampleLimit,
				enforcedTargetLimit,
				shards,
				secrets,
			)
			if err != nil {
				return nil, fmt.Errorf("failed generating service monitor %s/%s: %w", sMon.Namespace, sMon.Name, err)
			}
			scrapeConfigs = append(scrapeConfigs, gen)
		}
	}
	for _, pMon := range cfg.PodMonitors {
		for i, ep := range pMon.Spec.PodMetricsEndpoints {
			gen, err := generatePodMonitor(
				agentNS,
				pMon,
				&ep,
				i,
				apiServer,
				overrideHonorLabels,
				overrideHonorTimestamps,
				ignoreNamespaceSelectors,
				enforcedNamespaceLabel,
				enforcedSampleLimit,
				enforcedTargetLimit,
				shards,
				secrets,
			)
			if err != nil {
				return nil, fmt.Errorf("failed generating pod monitor %s/%s: %w", pMon.Namespace, pMon.Name, err)
			}
			scrapeConfigs = append(scrapeConfigs, gen)
		}
	}
	for _, probe := range cfg.Probes {
		gen, err := generateProbe(
			agentNS,
			probe,
			apiServer,
			overrideHonorLabels,
			overrideHonorTimestamps,
			ignoreNamespaceSelectors,
			enforcedNamespaceLabel,
			enforcedSampleLimit,
			enforcedTargetLimit,
			shards,
			secrets,
		)
		if err != nil {
			return nil, fmt.Errorf("failed generating service monitor %s/%s: %w", probe.Namespace, probe.Name, err)
		}
		scrapeConfigs = append(scrapeConfigs, gen)
	}

	if spec.AdditionalScrapeConfigs != nil {
		val, ok := secrets[assets.KeyForSecret(ns, spec.AdditionalScrapeConfigs)]
		if !ok {
			return nil, fmt.Errorf("expected secret for additionalScrapeConfigs")
		}

		var additionalScrapeConfigs []util.Map
		if err := yaml.Unmarshal([]byte(val), &additionalScrapeConfigs); err != nil {
			return nil, fmt.Errorf("failed unmarshaling additionalScrapeConfigs: %w", err)
		}
		scrapeConfigs = append(scrapeConfigs, additionalScrapeConfigs...)
	}

	ms.Set("scrape_configs", scrapeConfigs)

	if len(spec.RemoteWrite) > 0 {
		var remoteWrites []util.Map

		for i, rw := range spec.RemoteWrite {
			rwYAML, err := generateRemoteWrite(ns, &rw, secrets)
			if err != nil {
				return nil, fmt.Errorf("error generating remote_write %d: %w", i, err)
			}
			remoteWrites = append(remoteWrites, rwYAML)
		}

		ms.Set("remote_write", remoteWrites)
	}

	return
}

func generateServiceMonitor(
	agentNS string,
	cfg *prom.ServiceMonitor,
	ep *prom.Endpoint,
	i int,
	apiServer *prom.APIServerConfig,
	overrideHonorLabels bool,
	overrideHonorTimestamps bool,
	ignoreNamespaceSelectors bool,
	enforcedNamespaceLabel string,
	enforcedSampleLimit *uint64,
	enforcedTargetLimit *uint64,
	shards int32,
	secrets assets.SecretStore,
) (ms util.Map, err error) {

	ms.Set("job_name", fmt.Sprintf("serviceMonitor/%s/%s/%d", cfg.Namespace, cfg.Name, i))
	ms.Set("honor_labels", honorLabels(ep.HonorLabels, overrideHonorLabels))
	if honor := honorTimestamps(ep.HonorTimestamps, overrideHonorTimestamps); honor != nil {
		ms.Set("honor_timestamps", honor)
	}

	selectedNamespaces := getNamespacesFromNamespaceSelector(&cfg.Spec.NamespaceSelector, cfg.Namespace, ignoreNamespaceSelectors)
	kubeConfig, err := generateKubeSDConfig(agentNS, selectedNamespaces, apiServer, "endpoints", secrets)
	if err != nil {
		return nil, err
	}
	ms = append(ms, kubeConfig...)

	if ep.Interval != "" {
		ms.Set("scrape_interval", ep.Interval)
	}
	if ep.ScrapeTimeout != "" {
		ms.Set("scrape_timeout", ep.ScrapeTimeout)
	}
	if ep.Path != "" {
		ms.Set("metrics_path", ep.Path)
	}
	if ep.ProxyURL != nil {
		ms.Set("proxy_url", ep.ProxyURL)
	}
	if ep.Params != nil {
		ms.Set("params", ep.Params)
	}
	if ep.Scheme != "" {
		ms.Set("scheme", ep.Scheme)
	}
	if ep.TLSConfig != nil {
		ms.Set("tls_config", generateTLSConfig(cfg.Namespace, ep.TLSConfig))
	}
	if ep.BearerTokenFile != "" {
		ms.Set("bearer_token_file", ep.BearerTokenFile)
	}
	if ep.BearerTokenSecret.Name != "" {
		val, ok := secrets[assets.KeyForSecret(cfg.Namespace, &ep.BearerTokenSecret)]
		if !ok {
			return nil, fmt.Errorf("could not find secret for bearer_token in endpoint")
		}
		ms.Set("bearer_token", val)
	}
	if ep.BasicAuth != nil {
		username, ok := secrets[assets.KeyForSecret(cfg.Namespace, &ep.BasicAuth.Username)]
		if !ok {
			return nil, fmt.Errorf("couldn't find username secret for endpoint basic_auth")
		}
		password, ok := secrets[assets.KeyForSecret(cfg.Namespace, &ep.BasicAuth.Password)]
		if !ok {
			return nil, fmt.Errorf("couldn't find password secret for endpoint basic_auth")
		}
		ms.Set("basic_auth", util.Map{
			{Key: "username", Value: username},
			{Key: "password", Value: password},
		})
	}

	relabels := initRelabelings()

	// Filter targets by services selected by the monitor.

	// Exact label matches.
	var labelKeys []string
	for k := range cfg.Spec.Selector.MatchLabels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)

	for _, k := range labelKeys {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			Action:       "keep",
			SourceLabels: []string{"__meta_kubernetes_service_label_" + SanitizeLabelName(k)},
			Regex:        cfg.Spec.Selector.MatchLabels[k],
		}))
	}

	// Set based label matching. We have to map the valid relations
	// `In`, `NotIn`, `Exists`, and `DoesNotExist` into relabeling rules.
	for _, exp := range cfg.Spec.Selector.MatchExpressions {
		switch exp.Operator {
		case metav1.LabelSelectorOpIn:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_service_label_" + SanitizeLabelName(exp.Key)},
				Regex:        strings.Join(exp.Values, "|"),
			}))
		case metav1.LabelSelectorOpNotIn:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "drop",
				SourceLabels: []string{"__meta_kubernetes_service_label_" + SanitizeLabelName(exp.Key)},
				Regex:        strings.Join(exp.Values, "|"),
			}))
		case metav1.LabelSelectorOpExists:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_service_labelpresent_" + SanitizeLabelName(exp.Key)},
				Regex:        "true",
			}))
		case metav1.LabelSelectorOpDoesNotExist:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "drop",
				SourceLabels: []string{"__meta_kubernetes_service_labelpresent_" + SanitizeLabelName(exp.Key)},
				Regex:        "true",
			}))
		}
	}

	// Filter targets based on correct port for the endpoint.
	if ep.Port != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			Action:       "keep",
			SourceLabels: []string{"__meta_kubernetes_endpoint_port_name"},
			Regex:        ep.Port,
		}))
	} else if ep.TargetPort != nil {
		if ep.TargetPort.StrVal != "" {
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_pod_container_port_name"},
				Regex:        ep.TargetPort.String(),
			}))
		} else if ep.TargetPort.IntVal != 0 {
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_pod_container_port_number"},
				Regex:        ep.TargetPort.String(),
			}))
		}
	}

	// Relabel namespace, pod, and service labels into proper labels.
	relabels = append(relabels, []util.Map{
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_endpoint_address_target_kind", "__meta_kubernetes_endpoint_address_target_name"},
			Separator:    ";",
			Regex:        "Node;(.*)",
			Replacement:  "$1",
			TargetLabel:  "node",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_endpoint_address_target_kind", "__meta_kubernetes_endpoint_address_target_name"},
			Separator:    ";",
			Regex:        "Pod;(.*)",
			Replacement:  "$1",
			TargetLabel:  "pod",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_namespace"},
			TargetLabel:  "namespace",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_service_name"},
			TargetLabel:  "service",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_pod_name"},
			TargetLabel:  "pod",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_pod_container_name"},
			TargetLabel:  "container",
		}),
	}...)

	// Relabel targetLabels from Service onto the target.
	for _, l := range cfg.Spec.TargetLabels {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_service_label_" + SanitizeLabelName(l)},
			TargetLabel:  SanitizeLabelName(l),
			Regex:        "(.+)",
			Replacement:  "$1",
		}))
	}
	for _, l := range cfg.Spec.PodTargetLabels {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_pod_label_" + SanitizeLabelName(l)},
			TargetLabel:  SanitizeLabelName(l),
			Regex:        "(.+)",
			Replacement:  "$1",
		}))
	}

	// By default, generate a safe job name from the service name. We also keep
	// this around if a jobLabel is set in case the targets don't actually have a
	// value for it. A single service may potentially have multiple metrics
	// endpoints, therefore the endpoints labels is filled with the ports name or
	// (as a fallback) the port number.

	relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
		SourceLabels: []string{"__meta_kubernetes_service_name"},
		TargetLabel:  "job",
		Replacement:  "$1",
	}))
	if cfg.Spec.JobLabel != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_service_label_" + SanitizeLabelName(cfg.Spec.JobLabel)},
			TargetLabel:  "job",
			Regex:        "(.+)",
			Replacement:  "$1",
		}))
	}

	if ep.Port != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			TargetLabel: "endpoint",
			Replacement: ep.Port,
		}))
	} else if ep.TargetPort != nil && ep.TargetPort.String() != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			TargetLabel: "endpoint",
			Replacement: ep.TargetPort.String(),
		}))
	}

	if ep.RelabelConfigs != nil {
		for _, c := range ep.RelabelConfigs {
			relabels = append(relabels, generateRelabelConfig(c))
		}
	}

	// Because of security risks, whenever enforcedNamespaceLabel is set, we want
	// to append it to the relabel_configs as the last relabeling to ensure it
	// overrides all other relabelings.
	relabels = append(relabels, enforceNamespaceLabel(cfg.Namespace, enforcedNamespaceLabel)...)

	relabels = append(relabels, generateAddressShardingRelabelingRules(shards)...)
	ms.Set("relabel_configs", relabels)

	if cfg.Spec.SampleLimit > 0 || enforcedSampleLimit != nil {
		ms.Set("sample_limit", getLimit(cfg.Spec.SampleLimit, enforcedSampleLimit))
	}
	if cfg.Spec.TargetLimit > 0 || enforcedTargetLimit != nil {
		ms.Set("target_limit", getLimit(cfg.Spec.TargetLimit, enforcedTargetLimit))
	}

	if ep.MetricRelabelConfigs != nil {
		var metricRelabelings []util.Map
		for _, c := range ep.MetricRelabelConfigs {
			if c.TargetLabel != "" && enforcedNamespaceLabel != "" && c.TargetLabel == enforcedNamespaceLabel {
				continue
			}
			metricRelabelings = append(metricRelabelings, generateRelabelConfig(c))
		}
		ms.Set("metric_relabel_configs", metricRelabelings)
	}

	return
}

// honorLabels determines what value honor_labels should be have. If honor
// labels are overwritten, honor_labels will be forced to false.
func honorLabels(useHonorLabels, overrideHonorLabels bool) bool {
	if useHonorLabels && overrideHonorLabels {
		return false
	}
	return useHonorLabels
}

// honorTimestamps returns a *bool for the value honor_timestamps should be set
// to.
func honorTimestamps(useHonorTimestamps *bool, overrideHonorTimestamps bool) *bool {
	if useHonorTimestamps == nil && !overrideHonorTimestamps {
		return nil
	}

	var honor bool
	if useHonorTimestamps != nil {
		honor = *useHonorTimestamps
	}

	honor = honor && !overrideHonorTimestamps
	return &honor
}

// getNamespacesFromNamespaceSelector gets a list of namespaces to select based
// on the given namespace selector, default namespace, and whether namespace
// selectors should be ignored.
func getNamespacesFromNamespaceSelector(sel *prom.NamespaceSelector, ns string, ignoreSelectors bool) []string {
	switch {
	case ignoreSelectors:
		return []string{ns}
	case sel.Any:
		return []string{}
	case len(sel.MatchNames) == 0:
		// If no names are manually provided, the default is to look in the current
		// namespace.
		return []string{ns}
	default:
		return sel.MatchNames
	}
}

// generateKubeSDConfig generates kubernetes_sd_configs with some basic
// settings. ns must be set to the Agent namespace where the APIServerConfig is
// held.
func generateKubeSDConfig(
	ns string,
	namespaces []string,
	apiServer *prom.APIServerConfig,
	role string,
	secrets assets.SecretStore,
) (ms util.Map, err error) {
	var sdConfig util.Map

	sdConfig.Set("role", role)

	if len(namespaces) > 0 {
		sdConfig.Set("namespaces", util.Map{{Key: "names", Value: namespaces}})
	}
	if apiServer != nil {
		sdConfig.Set("api_server", apiServer.Host)
		if apiServer.BasicAuth != nil {
			username, ok := secrets[assets.KeyForSecret(ns, &apiServer.BasicAuth.Username)]
			if !ok {
				return nil, fmt.Errorf("couldn't find username secret for kubernetes_sd_configs basic_auth")
			}
			password, ok := secrets[assets.KeyForSecret(ns, &apiServer.BasicAuth.Password)]
			if !ok {
				return nil, fmt.Errorf("couldn't find password secret for kubernetes_sd_configs basic_auth")
			}
			sdConfig.Set("basic_auth", util.Map{
				{Key: "username", Value: username},
				{Key: "password", Value: password},
			})
		}
		if apiServer.BearerToken != "" {
			sdConfig.Set("bearer_token", apiServer.BearerToken)
		}
		if apiServer.BearerTokenFile != "" {
			sdConfig.Set("bearer_token_file", apiServer.BearerTokenFile)
		}
		if apiServer.TLSConfig != nil {
			sdConfig.Set("tls_config", generateTLSConfig(ns, apiServer.TLSConfig))
		}
	}

	ms.Set("kubernetes_sd_configs", []util.Map{sdConfig})
	return
}

func initRelabelings() []util.Map {
	return []util.Map{{
		{Key: "source_labels", Value: []string{"job"}},
		{Key: "target_label", Value: "__tmp_prometheus_job_name"},
	}}
}

// SanitizeLabelName sanitizes a label name for Prometheus.
func SanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}

func enforceNamespaceLabel(ns, enforcedNamespace string) []util.Map {
	if enforcedNamespace == "" {
		return nil
	}
	return []util.Map{
		generateRelabelConfig(&prom.RelabelConfig{
			TargetLabel: enforcedNamespace,
			Replacement: ns,
		}),
	}
}

func generateAddressShardingRelabelingRules(shards int32) []util.Map {
	return []util.Map{
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__address__"},
			TargetLabel:  "__tmp_hash",
			Modulus:      uint64(shards),
			Action:       "hashmod",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__tmp_hash"},
			Regex:        "${SHARD}",
			Action:       "keep",
		}),
	}
}

func getLimit(user uint64, enforced *uint64) uint64 {
	if enforced != nil {
		if user < *enforced && user != 0 || *enforced == 0 {
			return user
		}
		return *enforced
	}
	return user
}

func generatePodMonitor(
	agentNS string,
	cfg *prom.PodMonitor,
	ep *prom.PodMetricsEndpoint,
	i int,
	apiServer *prom.APIServerConfig,
	overrideHonorLabels bool,
	overrideHonorTimestamps bool,
	ignoreNamespaceSelectors bool,
	enforcedNamespaceLabel string,
	enforcedSampleLimit *uint64,
	enforcedTargetLimit *uint64,
	shards int32,
	secrets assets.SecretStore,
) (ms util.Map, err error) {

	ms.Set("job_name", fmt.Sprintf("podMonitor/%s/%s/%d", cfg.Namespace, cfg.Name, i))
	ms.Set("honor_labels", honorLabels(ep.HonorLabels, overrideHonorLabels))
	if honor := honorTimestamps(ep.HonorTimestamps, overrideHonorTimestamps); honor != nil {
		ms.Set("honor_timestamps", honor)
	}

	selectedNamespaces := getNamespacesFromNamespaceSelector(&cfg.Spec.NamespaceSelector, cfg.Namespace, ignoreNamespaceSelectors)
	kubeConfig, err := generateKubeSDConfig(agentNS, selectedNamespaces, apiServer, "pod", secrets)
	if err != nil {
		return nil, err
	}
	ms = append(ms, kubeConfig...)

	if ep.Interval != "" {
		ms.Set("scrape_interval", ep.Interval)
	}
	if ep.ScrapeTimeout != "" {
		ms.Set("scrape_timeout", ep.ScrapeTimeout)
	}
	if ep.Path != "" {
		ms.Set("metrics_path", ep.Path)
	}
	if ep.ProxyURL != nil {
		ms.Set("proxy_url", ep.ProxyURL)
	}
	if ep.Params != nil {
		ms.Set("params", ep.Params)
	}
	if ep.Scheme != "" {
		ms.Set("scheme", ep.Scheme)
	}
	if ep.TLSConfig != nil {
		ms.Set("tls_config", generateSafeTLSConfig(cfg.Namespace, &ep.TLSConfig.SafeTLSConfig))
	}
	if ep.BearerTokenSecret.Name != "" {
		val, ok := secrets[assets.KeyForSecret(cfg.Namespace, &ep.BearerTokenSecret)]
		if !ok {
			return nil, fmt.Errorf("could not find secret for bearer_token in endpoint")
		}
		ms.Set("bearer_token", val)
	}
	if ep.BasicAuth != nil {
		username, ok := secrets[assets.KeyForSecret(cfg.Namespace, &ep.BasicAuth.Username)]
		if !ok {
			return nil, fmt.Errorf("couldn't find username secret for endpoint basic_auth")
		}
		password, ok := secrets[assets.KeyForSecret(cfg.Namespace, &ep.BasicAuth.Password)]
		if !ok {
			return nil, fmt.Errorf("couldn't find password secret for endpoint basic_auth")
		}
		ms.Set("basic_auth", util.Map{
			{Key: "username", Value: username},
			{Key: "password", Value: password},
		})
	}

	relabels := initRelabelings()

	// Filter targets by pods selected by the monitor.
	// Exact label matches.
	var labelKeys []string
	for k := range cfg.Spec.Selector.MatchLabels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)

	for _, k := range labelKeys {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			Action:       "keep",
			SourceLabels: []string{"__meta_kubernetes_pod_label_" + SanitizeLabelName(k)},
			Regex:        cfg.Spec.Selector.MatchLabels[k],
		}))
	}

	// Set based label matching. We have to map the valid relations
	// `In`, `NotIn`, `Exists`, and `DoesNotExist` into relabeling rules.
	for _, exp := range cfg.Spec.Selector.MatchExpressions {
		switch exp.Operator {
		case metav1.LabelSelectorOpIn:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_pod_label_" + SanitizeLabelName(exp.Key)},
				Regex:        strings.Join(exp.Values, "|"),
			}))
		case metav1.LabelSelectorOpNotIn:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "drop",
				SourceLabels: []string{"__meta_kubernetes_pod_label_" + SanitizeLabelName(exp.Key)},
				Regex:        strings.Join(exp.Values, "|"),
			}))
		case metav1.LabelSelectorOpExists:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_pod_labelpresent_" + SanitizeLabelName(exp.Key)},
				Regex:        "true",
			}))
		case metav1.LabelSelectorOpDoesNotExist:
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "drop",
				SourceLabels: []string{"__meta_kubernetes_pod_labelpresent_" + SanitizeLabelName(exp.Key)},
				Regex:        "true",
			}))
		}
	}

	// Filter targets based on correct port for the endpoint.
	if ep.Port != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			Action:       "keep",
			SourceLabels: []string{"__meta_kubernetes_pod_container_port_name"},
			Regex:        ep.Port,
		}))
	} else if ep.TargetPort != nil {
		if ep.TargetPort.StrVal != "" {
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_pod_container_port_name"},
				Regex:        ep.TargetPort.String(),
			}))
		} else if ep.TargetPort.IntVal != 0 {
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_pod_container_port_number"},
				Regex:        ep.TargetPort.String(),
			}))
		}
	}

	// Relabel namespace, pod, and service labels into proper labels.
	relabels = append(relabels, []util.Map{
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_namespace"},
			TargetLabel:  "namespace",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_pod_name"},
			TargetLabel:  "pod",
		}),
		generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_pod_container_name"},
			TargetLabel:  "container",
		}),
	}...)

	for _, l := range cfg.Spec.PodTargetLabels {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_pod_label_" + SanitizeLabelName(l)},
			TargetLabel:  SanitizeLabelName(l),
			Regex:        "(.+)",
			Replacement:  "$1",
		}))
	}

	// By default, generate a safe job name from the PodMonitor. We also keep
	// this around if a jobLabel is set in case the targets don't actually have a
	// value for it. A single pod may potentially have multiple metrics
	// endpoints, therefore the endpoints labels is filled with the ports name or
	// (as a fallback) the port number.

	relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
		TargetLabel: "job",
		Replacement: fmt.Sprintf("%s/%s", cfg.GetNamespace(), cfg.GetName()),
	}))
	if cfg.Spec.JobLabel != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			SourceLabels: []string{"__meta_kubernetes_pod_label_" + SanitizeLabelName(cfg.Spec.JobLabel)},
			TargetLabel:  "job",
			Regex:        "(.+)",
			Replacement:  "$1",
		}))
	}

	if ep.Port != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			TargetLabel: "endpoint",
			Replacement: ep.Port,
		}))
	} else if ep.TargetPort != nil && ep.TargetPort.String() != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			TargetLabel: "endpoint",
			Replacement: ep.TargetPort.String(),
		}))
	}

	if ep.RelabelConfigs != nil {
		for _, c := range ep.RelabelConfigs {
			relabels = append(relabels, generateRelabelConfig(c))
		}
	}

	// Because of security risks, whenever enforcedNamespaceLabel is set, we want
	// to append it to the relabel_configs as the last relabeling to ensure it
	// overrides all other relabelings.
	relabels = append(relabels, enforceNamespaceLabel(cfg.Namespace, enforcedNamespaceLabel)...)

	relabels = append(relabels, generateAddressShardingRelabelingRules(shards)...)
	ms.Set("relabel_configs", relabels)

	if cfg.Spec.SampleLimit > 0 || enforcedSampleLimit != nil {
		ms.Set("sample_limit", getLimit(cfg.Spec.SampleLimit, enforcedSampleLimit))
	}
	if cfg.Spec.TargetLimit > 0 || enforcedTargetLimit != nil {
		ms.Set("target_limit", getLimit(cfg.Spec.TargetLimit, enforcedTargetLimit))
	}

	if ep.MetricRelabelConfigs != nil {
		var metricRelabelings []util.Map
		for _, c := range ep.MetricRelabelConfigs {
			if c.TargetLabel != "" && enforcedNamespaceLabel != "" && c.TargetLabel == enforcedNamespaceLabel {
				continue
			}
			metricRelabelings = append(metricRelabelings, generateRelabelConfig(c))
		}
		ms.Set("metric_relabel_configs", metricRelabelings)
	}

	return
}

func generateProbe(
	agentNS string,
	cfg *prom.Probe,
	apiServer *prom.APIServerConfig,
	overrideHonorLabels bool,
	overrideHonorTimestamps bool,
	ignoreNamespaceSelectors bool,
	enforcedNamespaceLabel string,
	enforcedSampleLimit *uint64,
	enforcedTargetLimit *uint64,
	shards int32,
	secrets assets.SecretStore,
) (ms util.Map, err error) {

	honorTS := true

	ms.Set("job_name", fmt.Sprintf("probe/%s/%s", cfg.Namespace, cfg.Name))
	if honor := honorTimestamps(&honorTS, overrideHonorTimestamps); honor != nil {
		ms.Set("honor_timestamps", honor)
	}

	path := "/probe"
	if cfg.Spec.ProberSpec.Path != "" {
		path = cfg.Spec.ProberSpec.Path
	}
	ms.Set("metrics_path", path)

	if cfg.Spec.Interval != "" {
		ms.Set("scrape_interval", cfg.Spec.Interval)
	}
	if cfg.Spec.ScrapeTimeout != "" {
		ms.Set("scrape_timeout", cfg.Spec.ScrapeTimeout)
	}
	if cfg.Spec.ProberSpec.Scheme != "" {
		ms.Set("scheme", cfg.Spec.ProberSpec.Scheme)
	}

	ms.Set("params", util.Map{
		{Key: "module", Value: []string{cfg.Spec.Module}},
	})

	relabels := initRelabelings()

	if cfg.Spec.JobName != "" {
		relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
			TargetLabel: "job",
			Replacement: cfg.Spec.JobName,
		}))
	}

	// Generate static_config section
	if cfg.Spec.Targets.StaticConfig != nil {
		var staticConfig util.Map
		staticConfig.Set("targets", cfg.Spec.Targets.StaticConfig.Targets)

		if cfg.Spec.Targets.StaticConfig.Labels != nil {
			if _, ok := cfg.Spec.Targets.StaticConfig.Labels["namespace"]; !ok {
				cfg.Spec.Targets.StaticConfig.Labels["namespace"] = cfg.Namespace
			}
		} else {
			cfg.Spec.Targets.StaticConfig.Labels = map[string]string{
				"namespace": cfg.Namespace,
			}
		}
		staticConfig.Set("labels", cfg.Spec.Targets.StaticConfig.Labels)

		ms.Set("static_configs", []util.Map{staticConfig})

		// Relabelings for prober.
		relabels = append(relabels, []util.Map{
			generateRelabelConfig(&prom.RelabelConfig{
				SourceLabels: []string{"__address__"},
				TargetLabel:  "__param_target",
			}),
			generateRelabelConfig(&prom.RelabelConfig{
				SourceLabels: []string{"__param_target"},
				TargetLabel:  "instance",
			}),
			generateRelabelConfig(&prom.RelabelConfig{
				TargetLabel: "__address__",
				Replacement: cfg.Spec.ProberSpec.URL,
			}),
		}...)

		// Add configured relabelings.
		for _, r := range cfg.Spec.Targets.StaticConfig.RelabelConfigs {
			relabels = append(relabels, generateRelabelConfig(r))
		}
	}

	// Generate kubernetes_sd_config section for ingress resources.
	if cfg.Spec.Targets.StaticConfig == nil {
		labelKeys := make([]string, 0, len(cfg.Spec.Targets.Ingress.Selector.MatchLabels))

		// Filter targets by ingresses selected by the monitor.
		// Exact label matches.
		for k := range cfg.Spec.Targets.Ingress.Selector.MatchLabels {
			labelKeys = append(labelKeys, k)
		}
		sort.Strings(labelKeys)

		for _, k := range labelKeys {
			relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
				Action:       "keep",
				SourceLabels: []string{"__meta_kubernetes_ingress_label_" + SanitizeLabelName(k)},
				Regex:        cfg.Spec.Targets.Ingress.Selector.MatchLabels[k],
			}))
		}

		// Set based label matching. We have to map the valid relations
		// `In`, `NotIn`, `Exists`, and `DoesNotExist` into relabeling rules.
		for _, exp := range cfg.Spec.Targets.Ingress.Selector.MatchExpressions {
			switch exp.Operator {
			case metav1.LabelSelectorOpIn:
				relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
					Action:       "keep",
					SourceLabels: []string{"__meta_kubernetes_ingress_label_" + SanitizeLabelName(exp.Key)},
					Regex:        strings.Join(exp.Values, "|"),
				}))
			case metav1.LabelSelectorOpNotIn:
				relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
					Action:       "drop",
					SourceLabels: []string{"__meta_kubernetes_ingress_label_" + SanitizeLabelName(exp.Key)},
					Regex:        strings.Join(exp.Values, "|"),
				}))
			case metav1.LabelSelectorOpExists:
				relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
					Action:       "keep",
					SourceLabels: []string{"__meta_kubernetes_ingress_labelpresent_" + SanitizeLabelName(exp.Key)},
					Regex:        "true",
				}))
			case metav1.LabelSelectorOpDoesNotExist:
				relabels = append(relabels, generateRelabelConfig(&prom.RelabelConfig{
					Action:       "drop",
					SourceLabels: []string{"__meta_kubernetes_ingress_labelpresent_" + SanitizeLabelName(exp.Key)},
					Regex:        "true",
				}))
			}
		}

		selectedNamespaces := getNamespacesFromNamespaceSelector(&cfg.Spec.Targets.Ingress.NamespaceSelector, cfg.Namespace, ignoreNamespaceSelectors)
		kubeConfig, err := generateKubeSDConfig(agentNS, selectedNamespaces, apiServer, "ingress", secrets)
		if err != nil {
			return nil, err
		}
		ms = append(ms, kubeConfig...)

		// Relabelings for ingress SD
		relabels = append(relabels, []util.Map{
			generateRelabelConfig(&prom.RelabelConfig{
				SourceLabels: []string{"__meta_kubernetes_ingress_scheme", "__address__", "__meta_kubernetes_ingress_path"},
				Separator:    ";",
				Regex:        "(.+);(.+);(.+)",
				TargetLabel:  "__param_target",
				Replacement:  "$1://$2$3",
				Action:       "replace",
			}),
			generateRelabelConfig(&prom.RelabelConfig{
				SourceLabels: []string{"__meta_kubernetes_namespace"},
				TargetLabel:  "namespace",
			}),
			generateRelabelConfig(&prom.RelabelConfig{
				SourceLabels: []string{"__meta_kubernetes_ingress_name"},
				TargetLabel:  "ingress",
			}),
		}...)

		// Relabelings for prober
		relabels = append(relabels, []util.Map{
			generateRelabelConfig(&prom.RelabelConfig{
				SourceLabels: []string{"__param_target"},
				TargetLabel:  "instance",
			}),
			generateRelabelConfig(&prom.RelabelConfig{
				TargetLabel: "__address__",
				Replacement: cfg.Spec.ProberSpec.URL,
			}),
		}...)

		// Add configured relabelings.
		for _, r := range cfg.Spec.Targets.Ingress.RelabelConfigs {
			relabels = append(relabels, generateRelabelConfig(r))
		}
	}

	relabels = append(relabels, enforceNamespaceLabel(cfg.Namespace, enforcedNamespaceLabel)...)
	ms.Set("relabel_configs", relabels)

	if cfg.Spec.TLSConfig != nil {
		ms.Set("tls_config", generateSafeTLSConfig(cfg.Namespace, &cfg.Spec.TLSConfig.SafeTLSConfig))
	}

	if cfg.Spec.BearerTokenSecret.Name != "" {
		val, ok := secrets[assets.KeyForSecret(cfg.Namespace, &cfg.Spec.BearerTokenSecret)]
		if !ok {
			return nil, fmt.Errorf("could not find secret for bearer_token in endpoint")
		}
		ms.Set("bearer_token", val)
	}
	if cfg.Spec.BasicAuth != nil {
		username, ok := secrets[assets.KeyForSecret(cfg.Namespace, &cfg.Spec.BasicAuth.Username)]
		if !ok {
			return nil, fmt.Errorf("couldn't find username secret for endpoint basic_auth")
		}
		password, ok := secrets[assets.KeyForSecret(cfg.Namespace, &cfg.Spec.BasicAuth.Password)]
		if !ok {
			return nil, fmt.Errorf("couldn't find password secret for endpoint basic_auth")
		}
		ms.Set("basic_auth", util.Map{
			{Key: "username", Value: username},
			{Key: "password", Value: password},
		})
	}

	return
}
