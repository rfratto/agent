// Package config generates Grafana Agent configuration based on Kubernetes
// resources.
package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/fatih/structs"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	grafana "github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
	"github.com/grafana/agent/pkg/operator/assets"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"gopkg.in/yaml.v2"
)

// Deployment is a set of resources used for one deployment of the Agent.
type Deployment struct {
	// Agent is the root resource that the deployment represents.
	Agent *grafana.GrafanaAgent
	// Prometheis is the set of prometheus instances discovered from the root Agent resource.
	Prometheis []PrometheusInstance
}

// BuildConfig builds an Agent configuration file.
func (d *Deployment) BuildConfig(secrets assets.SecretStore) (string, error) {
	vm := jsonnet.MakeVM()
	vm.StringOutput = true

	templatesContents, err := fs.Sub(templates, "templates")
	if err != nil {
		return "", err
	}

	vm.Importer(NewFSImporter(templatesContents))

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "marshalYAML",
		Params: ast.Identifiers{"object"},
		Func: func(i []interface{}) (interface{}, error) {
			bb, err := yaml.Marshal(i[0])
			if err != nil {
				return nil, jsonnet.RuntimeError{Msg: err.Error()}
			}
			return string(bb), nil
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "trimOptional",
		Params: ast.Identifiers{"value"},
		Func: func(i []interface{}) (interface{}, error) {
			m := i[0].(map[string]interface{})
			trimMap(m)
			return m, nil
		},
	})
	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "secretLookup",
		Params: ast.Identifiers{"key"},
		Func: func(i []interface{}) (interface{}, error) {
			if i[0] == nil {
				return nil, nil
			}

			k := assets.Key(i[0].(string))
			val, ok := secrets[k]
			if !ok {
				return nil, jsonnet.RuntimeError{Msg: fmt.Sprintf("key not provided: %s", k)}
			}
			return val, nil
		},
	})
	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "secretPath",
		Params: ast.Identifiers{"key"},
		Func: func(i []interface{}) (interface{}, error) {
			if i[0] == nil {
				return nil, nil
			}

			key := SanitizeLabelName(i[0].(string))
			return filepath.Join("/var/lib/grafana-agent/secrets", key), nil
		},
	})
	vm.NativeFunction(&jsonnet.NativeFunction{
		Name: "context",
		Func: func(i []interface{}) (interface{}, error) {
			return d, nil
		},
	})

	// Hack: we want to allow the Jsonnet code to reference the deployment's
	// fields using the Go names and NOT the JSON names.
	bb, err := json.Marshal(structs.Map(d))
	if err != nil {
		return "", err
	}

	vm.TLACode("ctx", string(bb))
	return vm.EvaluateFile("./agent.libsonnet")
}

// PrometheusInstance is an instance with a set of associated service monitors,
// pod monitors, and probes, which compose the final configuration of the
// generated Prometheus instance.
type PrometheusInstance struct {
	Instance        *grafana.PrometheusInstance
	ServiceMonitors []*prom.ServiceMonitor
	PodMonitors     []*prom.PodMonitor
	Probes          []*prom.Probe
}
