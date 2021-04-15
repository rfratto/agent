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
		store     assets.SecretStore
	)

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
					url: http://localhost:9090/api/prom/push
			`),
			expect: util.Untab(`
				scrape_interval: 15s
				scrape_timeout: 10s
				external_labels:
					cluster: prod
				remote_write:
				- name: rw-1
					url: http://localhost:9090/api/prom/push
			`),
		},

		// TODO(rfratto): more tests. ensure that secrets are loaded for
		// remote_write for all fields that don't support files.
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
