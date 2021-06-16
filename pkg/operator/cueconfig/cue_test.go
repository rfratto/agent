package config

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestCue(t *testing.T) {
	set := load.Instances([]string{"."}, &load.Config{
		Dir: "./templates",
	})
	for _, s := range set {
		require.NoError(t, s.Complete())

		ctx := cuecontext.New()

		v := ctx.BuildInstance(s)
		v = v.FillPath(cue.ParsePath("#Input"), &Deployment{
			Agent: &v1alpha1.GrafanaAgent{
				Spec: v1alpha1.GrafanaAgentSpec{
					LogLevel: "debug",
				},
			},
		})

		bb, err := v.MarshalJSON()
		require.NoError(t, err)

		var m yaml.MapSlice
		err = yaml.Unmarshal(bb, &m)
		require.NoError(t, err)

		bb, err = yaml.Marshal(m)
		require.NoError(t, err)
		fmt.Println(string(bb))
	}
}
