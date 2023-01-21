package prometheusconvert

import (
	"bytes"
	"fmt"

	"github.com/go-kit/log"
	"github.com/grafana/agent/converter/internal/schema/riverschema"
	"github.com/grafana/agent/pkg/river/token/builder"
	promconfig "github.com/prometheus/prometheus/config"
	_ "github.com/prometheus/prometheus/discovery/install" // Register Prometheus SDs
)

// Convert implements a Prometheus config converter.
func Convert(in []byte) ([]byte, error) {
	inConfig, err := promconfig.Load(string(in), false, log.NewNopLogger())
	if err != nil {
		return nil, err
	}
	f := builder.NewFile()

	// TODO(rfratto): providing *extremely* basic examples below to get a feel
	// for how this is going to work. The basic gist seems like it's on the right
	// track though, and now it's only a matter of writing the schema generator
	// so we can get type safety going.
	{
		scrape := builder.NewBlock([]string{"prometheus", "scrape"}, "prometheus")
		scrape.Body().SetAttributeValue("targets", []map[string]string{
			{"__address__": "localhost:9090"},
		})
		scrape.Body().SetAttributeValue("forward_to", []*riverschema.Capsule{
			riverschema.ExprCapsule("prometheus.remote_write.default.receiver"),
		})

		f.Body().AppendBlock(scrape)
	}

	{
		rw := builder.NewBlock([]string{"prometheus", "remote_write"}, "default")

		for _, rwConf := range inConfig.RemoteWriteConfigs {
			endpoint := builder.NewBlock([]string{"endpoint"}, "")
			endpoint.Body().SetAttributeValue("url", rwConf.URL.String())
			rw.Body().AppendBlock(endpoint)
		}

		f.Body().AppendBlock(rw)
	}

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("failed to render Flow config: %w", err)
	}
	return buf.Bytes(), nil
}
