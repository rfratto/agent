package metrics

import (
	"sort"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/grafana/agent/pkg/agentflow/config"
	"github.com/grafana/agent/pkg/agentflow/types/actorstate"
	"github.com/grafana/agent/pkg/agentflow/types/exchange"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
)

type MetricRelabel struct {
	name  string
	cfg   config.MetricRelabel
	rules []*relabel.Config

	self *actor.PID
	outs []*actor.PID
}

func NewRelabel(name string, cfg config.MetricRelabel) (actorstate.FlowActor, error) {
	promRules := make([]*relabel.Config, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		promRules = append(promRules, &relabel.Config{
			SourceLabels: toLabelNames(rule.SourceLabels),
			Separator:    rule.Separator,
			Regex:        relabel.MustNewRegexp(rule.Regex.String()),
			Modulus:      rule.Modulus,
			TargetLabel:  rule.TargetLabel,
			Replacement:  rule.Replacement,
			Action:       relabel.Action(rule.Action),
		})
	}

	return &MetricRelabel{
		name:  name,
		cfg:   cfg,
		rules: promRules,
	}, nil
}

func toLabelNames(in []string) model.LabelNames {
	res := make(model.LabelNames, len(in))
	for i, v := range in {
		res[i] = model.LabelName(v)
	}
	sort.Sort(res)
	return res
}

func (m *MetricRelabel) AllowableInputs() []actorstate.InOutType {
	return []actorstate.InOutType{actorstate.Metrics}
}

func (m *MetricRelabel) Output() actorstate.InOutType {
	return actorstate.Metrics
}

func (m *MetricRelabel) Name() string { return m.name }

func (m *MetricRelabel) PID() *actor.PID { return m.self }

func (m *MetricRelabel) Receive(c actor.Context) {
	switch msg := c.Message().(type) {
	case actorstate.Init:
		m.outs = msg.Children
	case actorstate.Start:
		m.self = c.Self()
	case []exchange.Metric:
		res := relabelMetrics(msg, m.rules)
		for _, out := range m.outs {
			c.Send(out, res)
		}
	}
}

func relabelMetrics(mm []exchange.Metric, rules []*relabel.Config) []exchange.Metric {
	out := make([]exchange.Metric, 0, len(mm))

	for _, m := range mm {
		promLabels := toPromLabels(m.Labels())
		promLabels = relabel.Process(promLabels, rules...)

		if promLabels == nil {
			// Metric is dropped
			continue
		}

		out = append(out, exchange.NewMetricFromPromMetric(
			m.Timestamp().UnixMilli(),
			m.Value(),
			promLabels,
		))
	}

	return out
}

func toPromLabels(in []exchange.Label) labels.Labels {
	out := make(labels.Labels, len(in))
	for _, label := range in {
		out = append(out, labels.Label{
			Name:  label.Key,
			Value: label.Value,
		})
	}
	return out
}

func fromPromLabels(in labels.Labels) []exchange.Label {
	out := make([]exchange.Label, len(in))
	for _, label := range in {
		out = append(out, exchange.Label{
			Key:   label.Name,
			Value: label.Value,
		})
	}
	return out
}
