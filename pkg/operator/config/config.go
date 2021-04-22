// Package config generates Grafana Agent configuration based on Kubernetes
// resources.
package config

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	grafana "github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
	"github.com/grafana/agent/pkg/operator/assets"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
)

// Deployment is a set of resources used for one deployment of the Agent.
type Deployment struct {
	// Agent is the root resource that the deployment represents.
	Agent *grafana.GrafanaAgent
	// Prometheis is the set of prometheus instances discovered from the root Agent resource.
	Prometheis []PrometheusInstance
	// Secrets are used for secret lookup for any configuration fields that
	// cannot be loaded directly from a file. If a secret is used, lookup must
	// succeed otherwise building will fail.
	Secrets assets.SecretStore
}

// BuildConfig builds an Agent configuration file.
func (d *Deployment) BuildConfig() (string, error) {
	tmpls := template.New("")
	tmpls.Option("missingkey=invalid")

	tmpls.Funcs(sprig.TxtFuncMap())
	tmpls.Funcs(template.FuncMap{
		"include": func(template string, v interface{}) (string, error) {
			tmpl := tmpls.Lookup(template)
			if tmpl == nil {
				return "", fmt.Errorf("no such template %s", template)
			}

			var sw strings.Builder
			if err := tmpl.Execute(&sw, v); err != nil {
				return "", err
			}
			return sw.String(), nil
		},

		"yaml": func(v interface{}) (string, error) {
			bb, err := yaml.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(bb), nil
		},
		"yamlField": yamlField,

		"namespacesFromSelector": func(sel *prom.NamespaceSelector, ns string, ignoreSelectors bool) []string {
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
		},
		"honorLabels": func(honor, override bool) bool {
			if honor && override {
				return false
			}
			return honor
		},
		"honorTimestamps": func(honor *bool, override bool) *bool {
			if honor == nil && !override {
				return nil
			}

			var val bool
			if honor != nil {
				val = *honor
			}

			val = val && !override
			return &val
		},

		"secretPath": func(ns string, sel interface{}) (string, error) {
			if sel == nil {
				return "", nil
			}

			switch v := sel.(type) {
			case prom.SecretOrConfigMap:
				if v.ConfigMap == nil && v.Secret == nil {
					return "", nil
				}
				return pathForKey(assets.KeyForSelector(ns, &v)), nil
			case *v1.SecretKeySelector:
				if v == nil {
					return "", nil
				}
				return pathForKey(assets.KeyForSecret(ns, v)), nil
			case *v1.ConfigMapKeySelector:
				if v == nil {
					return "", nil
				}
				return pathForKey(assets.KeyForConfigMap(ns, v)), nil
			default:
				return "", fmt.Errorf("unknown secretPath type %T", v)
			}
		},
		"secretValue": func(ns string, sel *v1.SecretKeySelector) (string, error) {
			if sel == nil {
				return "", nil
			}

			path := assets.KeyForSecret(ns, sel)
			val, ok := d.Secrets[path]
			if !ok {
				return "", fmt.Errorf("no secret: %s", path)
			}
			return val, nil
		},
	})

	err := fs.WalkDir(templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		bb, err := fs.ReadFile(templates, path)
		if err != nil {
			return err
		}

		relative := strings.TrimPrefix(path, "templates/")
		_, err = tmpls.New(relative).Parse(string(bb))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error reading templates: %w", err)
	}

	var sw strings.Builder
	if err := tmpls.Lookup("agent.yaml").Execute(&sw, d); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return reformatYAML(sw.String())
}

func reformatYAML(in string) (string, error) {
	var raw yaml.MapSlice
	if err := yaml.Unmarshal([]byte(in), &raw); err != nil {
		return "", err
	}
	bb, err := yaml.Marshal(raw)
	return string(bb), err
}

// PrometheusInstance is an instance with a set of associated service monitors,
// pod monitors, and probes, which compose the final configuration of the
// generated Prometheus instance.
type PrometheusInstance struct {
	Instance        *grafana.PrometheusInstance
	ServiceMonitors []*prom.ServiceMonitor
	PodMonitors     []*prom.PodMonitor
	Probes          []*prom.Probe

	// AdditionalScrapeConfigs are user-configured scrape configs to manually
	// add into the file.
	AdditionalScrapeConfigs []yaml.MapSlice
}

func pathForKey(key assets.Key) string {
	return filepath.Join("/var/lib/grafana-agent/secrets", SanitizeLabelName(string(key)))
}

var invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// SanitizeLabelName sanitizes a label name for Prometheus.
func SanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}
