package discoveryext

import (
	"encoding/json"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

// TargetGroup implements json.Marshaler for targetgroup.Group, marshaling it
// in the expected http_sd format.
type TargetGroup targetgroup.Group

func (tg *TargetGroup) MarshalJSON() ([]byte, error) {
	g := &struct {
		Targets []string       `json:"targets"`
		Labels  model.LabelSet `json:"labels,omitempty"`
	}{
		Targets: make([]string, 0, len(tg.Targets)),
		Labels:  tg.Labels,
	}
	for _, t := range tg.Targets {
		g.Targets = append(g.Targets, string(t[model.AddressLabel]))
	}
	return json.Marshal(g)
}
