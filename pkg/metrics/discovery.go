package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/agent/pkg/metrics/internal/metricspb"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

// discoveryJob performs Prometheus SD and distributes discovered jobs
// throughout the cluster.
type discoveryJob struct {
	key string
	log log.Logger

	wg     *sync.WaitGroup
	cancel context.CancelFunc

	mgr  *discovery.Manager
	dist *distributor
}

func newDiscoveryJob(l log.Logger, key string, dist *distributor, sc []*config.ScrapeConfig) (*discoveryJob, error) {
	ctx, cancel := context.WithCancel(context.Background())

	l = log.With(l, "component", "metrics-target-discovery", "instance", key)
	mgr := discovery.NewManager(ctx, l, discovery.Name(key))

	var wg sync.WaitGroup

	dj := &discoveryJob{
		key: key,
		log: l,

		wg:     &wg,
		cancel: cancel,

		mgr:  mgr,
		dist: dist,
	}
	if err := dj.ApplyConfig(sc); err != nil {
		return nil, err
	}

	wg.Add(2)
	go dj.runDiscovery()
	go dj.runTargetSharding(ctx)
	return dj, nil
}

// ApplyConfig updates the set of jobs to perform SD for.
func (d *discoveryJob) ApplyConfig(sc []*config.ScrapeConfig) error {
	dc := map[string]discovery.Configs{}
	for _, c := range sc {
		dc[c.JobName] = c.ServiceDiscoveryConfigs
	}
	if err := d.mgr.ApplyConfig(dc); err != nil {
		return fmt.Errorf("failed to update the discovery manager: %w", err)
	}
	return nil
}

func (d *discoveryJob) Close() error {
	defer d.wg.Wait()
	d.cancel()
	return nil
}

func (d *discoveryJob) runDiscovery() {
	defer d.wg.Done()

	// Context already passed to manager in newDiscoveryJob.
	err := d.mgr.Run()
	if err != nil {
		level.Error(d.log).Log("msg", "discovery manager stopped with error", "err", err)
	} else {
		level.Debug(d.log).Log("msg", "discovery manager stopped")
	}
}

func (d *discoveryJob) runTargetSharding(ctx context.Context) {
	defer d.wg.Done()
	defer level.Debug(d.log).Log("msg", "target sharding stopped")

	// TODO(rfratto): configurable TTL?
	ttl := time.Minute * 3
	shardTimeout := ttl / 4

	// Cache of previous syncs.
	var (
		// Used for calculating tombstones
		prev = make(map[string][]*targetgroup.Group)
		// Used for refreshing TTL.
		lastReq *metricspb.ScrapeTargetsRequest
	)

Loop:
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(ttl / 2):
			if lastReq == nil {
				continue Loop
			}

			level.Info(d.log).Log("msg", "forcing new target distribution")

			// Update the DiscoveryTime so the TTL is refreshed.
			lastReq.DiscoveryTime = time.Now().UnixNano()

			ctx, cancel := context.WithTimeout(context.Background(), shardTimeout)
			_, _ = d.dist.ScrapeTargets(ctx, lastReq)
			cancel()
		case tgs := <-d.mgr.SyncCh():
			ctx, cancel := context.WithTimeout(context.Background(), shardTimeout)

			shardReq := d.buildShardRequest(prev, tgs, ttl)
			_, _ = d.dist.ScrapeTargets(ctx, shardReq)

			// Update cache.
			prev = tgs
			lastReq = shardReq

			cancel()
		}
	}
}

func (d *discoveryJob) buildShardRequest(prev, cur map[string][]*targetgroup.Group, ttl time.Duration) *metricspb.ScrapeTargetsRequest {
	req := &metricspb.ScrapeTargetsRequest{
		InstanceName:  d.key,
		Targets:       make(map[string]*metricspb.TargetSet),
		Tombstones:    make(map[string]*metricspb.TargetSet),
		DiscoveryTime: time.Now().UnixNano(),
		Ttl:           int64(ttl),
	}

	// Easy part: copy over targets from cur.
	for setName, groups := range cur {
		tset := metricspb.GetTargetSet(req, setName)

		for _, group := range groups {
			tgroup := &metricspb.TargetGroup{
				Targets: fromLabelSets(group.Targets),
				Labels:  fromLabelSet(group.Labels),
				Source:  group.Source,
			}
			tset.Groups = append(tset.Groups, tgroup)
		}
	}

	// Hard part: any target from prev that is not in cur should be tombstoned.
	for setName, groups := range prev {
		for _, group := range groups {
			var curGroup *targetgroup.Group
			for _, g := range cur[setName] {
				if g.Source == group.Source {
					curGroup = g
					break
				}
			}
			for _, target := range group.Targets {
				tombstoneTarget := true
				var curTargets []model.LabelSet
				if curGroup != nil {
					curTargets = curGroup.Targets
				}
				for _, t := range curTargets {
					if t.Equal(target) {
						tombstoneTarget = false
						break
					}
				}

				if tombstoneTarget {
					tset := metricspb.GetTombstones(req, setName)
					tgroup := metricspb.GetTargetGroup(tset, &metricspb.TargetGroup{
						Labels: fromLabelSet(group.Labels),
						Source: group.Source,
					})
					tgroup.Targets = append(tgroup.Targets, fromLabelSet(target))
				}
			}
		}
	}

	return req
}

func fromLabelSets(lsets []model.LabelSet) []*metricspb.LabelSet {
	res := make([]*metricspb.LabelSet, 0, len(lsets))
	for _, lset := range lsets {
		res = append(res, fromLabelSet(lset))
	}
	return res
}

func fromLabelSet(lset model.LabelSet) *metricspb.LabelSet {
	res := &metricspb.LabelSet{Labels: make(map[string]string, len(lset))}
	for k, v := range lset {
		res.Labels[string(k)] = string(v)
	}
	return res
}
