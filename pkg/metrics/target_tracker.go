package metrics

import (
	"context"
	"sync"

	"github.com/grafana/agent/pkg/metrics/internal/metricspb"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

// targetTracker manages a set of sharded targets.
type targetTracker struct {
	ch chan map[string][]*targetgroup.Group

	mut   sync.Mutex
	cache map[string][]*targetgroup.Group
}

// newTargetTracker creates a new targetTracker.
func newTargetTracker() *targetTracker {
	return &targetTracker{
		ch:    make(chan map[string][]*targetgroup.Group),
		cache: make(map[string][]*targetgroup.Group),
	}
}

// SyncCh returns a channel where targets are written whenever the set of
// targets changes.
func (tt *targetTracker) SyncCh() <-chan map[string][]*targetgroup.Group {
	return tt.ch
}

// Track updates the set of targets. req must contain the full set of
// targets that are expected to be scraped.
func (tt *targetTracker) Track(ctx context.Context, req *metricspb.ScrapeTargetsRequest) (totalTargets int, err error) {
	tt.mut.Lock()
	defer tt.mut.Unlock()

	tt.cache = make(map[string][]*targetgroup.Group)

	for group, tset := range req.GetTargets() {
		storeGroups := tt.cache[group]

		for _, tgroup := range tset.GetGroups() {
			storeGroup := &targetgroup.Group{
				Source:  tgroup.GetSource(),
				Targets: toLabelSets(tgroup.GetTargets()),
				Labels:  toLabelSet(tgroup.GetLabels()),
			}
			storeGroups = append(storeGroups, storeGroup)
			tt.cache[group] = storeGroups
		}
	}

	for _, tset := range tt.cache {
		for _, tgroup := range tset {
			totalTargets += len(tgroup.Targets)
		}
	}

	select {
	case tt.ch <- tt.cache:
		return totalTargets, nil
	case <-ctx.Done():
		return totalTargets, ctx.Err()
	}
}

func toLabelSets(lss []*metricspb.LabelSet) []model.LabelSet {
	res := make([]model.LabelSet, 0, len(lss))
	for _, ls := range lss {
		res = append(res, toLabelSet(ls))
	}
	return res
}

func toLabelSet(ls *metricspb.LabelSet) model.LabelSet {
	res := make(model.LabelSet, len(ls.GetLabels()))
	for k, v := range ls.GetLabels() {
		res[model.LabelName(k)] = model.LabelValue(v)
	}
	return res
}
