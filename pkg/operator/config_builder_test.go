package operator

import (
	"fmt"
	"testing"

	"github.com/grafana/agent/pkg/operator/assets"
	"github.com/grafana/agent/pkg/util"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	k8s_yaml "sigs.k8s.io/yaml"

	grafana "github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
)

func Test_buildPrometheusGlobal(t *testing.T) {
	// Shared objects across all tests
	var (
		namespace = "default"
		store     = make(assets.SecretStore)
	)

	store[assets.Key("/secrets/default/example-secret/key")] = "somesecret"
	store[assets.Key("/configMaps/default/example-cm/key")] = "somecm"

	tt := []struct {
		input  string
		expect string
	}{
		{
			input: util.Untab(`
				scrapeInterval: 15s
				scrapeTimeout: 10s
				externalLabels:
					cluster: prod
				remoteWrite:
				- name: rw-1
					url: http://localhost:9090/api/v1/write
			`),
			expect: util.Untab(`
				scrape_interval: 15s
				scrape_timeout: 10s
				external_labels:
					cluster: prod
				remote_write:
				- name: rw-1
					url: http://localhost:9090/api/v1/write
			`),
		},

		{
			input: util.Untab(`
				remoteWrite:
				- url: http://localhost:9090/api/v1/write
					basicAuth:
						username:
							name: example-secret
							key: key
						password:
							name: example-secret
							key: key
					tlsConfig:
						ca:
							configMap:
								name:	example-cm
								key: key
						cert:
							secret:
								name: example-secret
								key: key
			`),
			expect: util.Untab(`
				remote_write:
				- url: http://localhost:9090/api/v1/write
					basic_auth:
						username: somesecret
						password: somesecret
					tls_config:
						ca_file: /var/lib/grafana-agent/secrets/_configMaps_default_example_cm_key
						cert_file: /var/lib/grafana-agent/secrets/_secrets_default_example_secret_key
						insecure_skip_verify: false
			`),
		},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("index_%d", i), func(t *testing.T) {
			var spec grafana.PrometheusSubsystemSpec
			err := k8s_yaml.Unmarshal([]byte(tc.input), &spec)
			require.NoError(t, err)

			resultMap, err := buildPrometheusGlobal(namespace, &spec, store)
			require.NoError(t, err)

			bb, err := yaml.Marshal(resultMap)
			require.NoError(t, err)

			require.YAMLEq(t, tc.expect, string(bb))
		})
	}
}
