package metrics

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/agent/pkg/cluster"
	"github.com/grafana/agent/pkg/metrics/internal/metricspb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/rfratto/croissant/node"
	"go.uber.org/atomic"
)

// TODO(rfratto): targets ttl?

// distributor performs target sharding and manages local scrape jobs.
// distributors receive targets in two ways: via the local process
// and via other nodes in the cluster.
//
// If the local process is running service discovery for any job, it will
// send all discovered targets to the local distributor for distribution.
// When distributors receive these targets, they should forward targets
// that they are not responsible for to the closest peer that the local
// process knows about.
//
// The distributor may receive targets from remote nodes. It should
// repeat the same process, forwarding any targets that they are not
// responsible for to the next closest peer that is. This process will
// repeat for each target until it has found its appropriate owner.
//
// To optimize on network utilization, targets are batched into the
// next appropriate node instead of handling them one by one. The ID
// of the next hop in routing is used for identifying the batch.
//
// Service discovery will generate tombstones of targets that have
// gone away from the previous scrape. This is not used for any reason
// other than telling a node that it should refresh its targets.
type distributor struct {
	metricspb.UnimplementedScraperServer

	reg     prometheus.Registerer
	cfg     Config
	log     log.Logger
	cluster *cluster.Cluster

	// Cache of the local targets. Updated when ScrapeTargets is called.
	// Keyed by metricspb.ScrapeTargetsRequest.InstanceName.
	localMut sync.Mutex
	local    map[string]*metricspb.ScrapeTargetsRequest

	scraperMut sync.Mutex
	scrapers   map[string]*scraper
	closed     atomic.Bool
}

func newDistributor(reg prometheus.Registerer, log log.Logger, cfg Config, cluster *cluster.Cluster) (*distributor, error) {
	dist := &distributor{
		reg:     reg,
		cfg:     cfg,
		log:     log,
		cluster: cluster,

		local:    make(map[string]*metricspb.ScrapeTargetsRequest),
		scrapers: make(map[string]*scraper),
	}
	if err := dist.ApplyConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to create target sharder: %w", err)
	}
	return dist, nil
}

// ScrapeTargets appends new targets to the distributor. Targets not belonging
// to it will be forwarded, otherwise local scrapers will be updated to start
// collecting from those targets.
func (d *distributor) ScrapeTargets(ctx context.Context, req *metricspb.ScrapeTargetsRequest) (*metricspb.ScrapeTargetsResponse, error) {
	// Pushing a new req is the same as resharding, except the local targets are
	// merged before redistributing.
	d.localMut.Lock()
	dest := d.local[req.InstanceName]
	merged := merge(dest, req)
	d.local[req.InstanceName] = merged
	d.localMut.Unlock()

	// Don't hold the lock here: it's possible for another node to call back our
	// own ScrapeTargets if it's shutting down.
	d.reshardInstance(ctx, merged)
	return &metricspb.ScrapeTargetsResponse{}, nil
}

func (d *distributor) reshardInstance(ctx context.Context, req *metricspb.ScrapeTargetsRequest) {
	// Create the paritions for sharding the targets based on the next ID
	// along the Pastry routing path.
	partitions := make(metricspb.RequestParititions)

	// Create a local partition. We need to do this so we can merge
	// failed propagations back locally at the end of the function.
	_ = metricspb.GetPartition(partitions, d.cluster.ID(), req)

	for group, tset := range req.Targets {
		for _, tgroup := range tset.Groups {
			for _, target := range tgroup.Targets {
				key := getKey(target.Labels[model.AddressLabel])
				next, _, err := d.cluster.Node().NextPeer(key)

				// Assign locally by default. Ensures that a target will always be
				// scraped even when there's a routing issue.
				nextID := d.cluster.ID()
				if err == nil && next.Addr != "" {
					nextID = next.ID
				}

				partition := metricspb.GetPartition(partitions, nextID, req)
				partitionTset := metricspb.GetTargetSet(partition, group)
				partitionTgroup := metricspb.GetTargetGroup(partitionTset, tgroup)
				partitionTgroup.Targets = append(partitionTgroup.Targets, target)
			}
		}
	}

	// Repeat the same process for tombstones, only needed to ensure that
	// a node will update its local scraper.
	for group, tset := range req.Targets {
		for _, tgroup := range tset.Groups {
			for _, target := range tgroup.Targets {
				key := getKey(target.Labels[model.AddressLabel])
				next, _, err := d.cluster.Node().NextPeer(key)

				// Assign locally by default. Ensures that a target will always be
				// scraped even when there's a routing issue.
				nextID := d.cluster.ID()
				if err == nil && next.Addr != "" {
					nextID = next.ID
				}

				// NOTE(rfratto): GetTombstones *must* be used below instead of
				// GetTargetSet.
				partition := metricspb.GetPartition(partitions, nextID, req)
				partitionTset := metricspb.GetTombstones(partition, group)
				partitionTgroup := metricspb.GetTargetGroup(partitionTset, tgroup)
				partitionTgroup.Targets = append(partitionTgroup.Targets, target)
			}
		}
	}

	// Deliver each partition. If a partition fails to get delivered, we'll
	// merge it into our local set.

	// Don't allow routing to ourselves so we can merge groups back in.
	cc := node.NewClient(d.cluster.Node(), node.WithAllowSelfRouting(false))
	cli := metricspb.NewScraperClient(cc)

	for nodeID, partition := range partitions {
		if nodeID == d.cluster.ID() {
			// Skip ourselves for now and handle at the end.
			continue
		}

		_, err := cli.ScrapeTargets(node.WithClientKey(ctx, nodeID), partition)
		if err != nil {
			level.Warn(d.log).Log("msg", "failed to send group to any node, taking responsibility", "err", err)
			partitions[d.cluster.ID()] = merge(partitions[d.cluster.ID()], partition)
		}
	}

	d.scrapeLocalTargets(ctx, partitions[d.cluster.ID()])
}

// merge combines two metricspb.ScrapeTargetsRequests.
//
// If src has a newer discovery time, dest = src.
// If src has an oldest discovery time, no-op (dest = dest).
// If src has an equal discovery time, dest += src.
func merge(dest, src *metricspb.ScrapeTargetsRequest) *metricspb.ScrapeTargetsRequest {
	if dest != nil && dest.InstanceName != src.InstanceName {
		panic("dest and src have different instance names")
	}

	if dest != nil && src.DiscoveryTime < dest.DiscoveryTime {
		return dest
	}

	merged := &metricspb.ScrapeTargetsRequest{
		InstanceName:  src.InstanceName,
		Targets:       make(map[string]*metricspb.TargetSet),
		Tombstones:    make(map[string]*metricspb.TargetSet),
		DiscoveryTime: src.DiscoveryTime,
		Ttl:           src.Ttl,
	}

	// Copy over the old jobs if dest has the same timestamp.
	if dest != nil && src.DiscoveryTime == dest.DiscoveryTime {
		mergeTargets(merged.Targets, dest.Targets)
		mergeTargets(merged.Tombstones, dest.Tombstones)
	}

	// Copy over new data
	mergeTargets(merged.Targets, src.Targets)
	mergeTargets(merged.Tombstones, src.Tombstones)
	return merged
}

func mergeTargets(dest, src map[string]*metricspb.TargetSet) {
	for k, srcTset := range src {
		destTset, ok := dest[k]
		if !ok {
			destTset = &metricspb.TargetSet{}
			dest[k] = destTset
		}

		for _, srcGroup := range srcTset.Groups {
			destGroup := metricspb.GetTargetGroup(destTset, srcGroup)
			destGroup.Targets = append(destGroup.Targets, srcGroup.GetTargets()...)
		}
	}
}

// Reshard redistributes the set of local targets.
func (d *distributor) Reshard(ctx context.Context) error {
	level.Info(d.log).Log("msg", "redistributing targets")
	defer level.Info(d.log).Log("msg", "finished redistributing targets")

	d.localMut.Lock()
	locals := make([]*metricspb.ScrapeTargetsRequest, 0, len(d.local))
	for _, req := range d.local {
		locals = append(locals, req)
	}
	d.localMut.Unlock()

	// Don't hold the lock here: it's possible for another node to call back our
	// own ScrapeTargets if it's shutting down.
	for _, req := range locals {
		d.reshardInstance(ctx, req)
	}
	return nil
}

// scrapeLocalTargets manages scrapers fro the local node. Targets found in req
// contain the full set of targets that should be scraped.
func (d *distributor) scrapeLocalTargets(ctx context.Context, req *metricspb.ScrapeTargetsRequest) {
	d.scraperMut.Lock()
	defer d.scraperMut.Unlock()

	if d.closed.Load() {
		return
	}

	// Launch a new scraper if we need to.
	sc, ok := d.scrapers[req.InstanceName]
	if !ok {
		if metricspb.CountTargets(req) == 0 {
			// Nothing to do here, there's no targets.
			return
		}

		var err error
		sc, err = newScraper(d.log, d.reg, d.cfg, req.InstanceName)
		if err != nil {
			level.Error(d.log).Log("msg", "unable to create scrape jobs", "err", err)
			return
		}
		d.scrapers[req.InstanceName] = sc
	}

	// Update the scraper's set of targets.
	totalTargets, err := sc.Track(ctx, req)
	if err != nil {
		level.Error(d.log).Log("msg", "unable to track scrape targets", "instance", req.InstanceName, "err", err)
		return
	} else if totalTargets == 0 {
		level.Debug(d.log).Log("msg", "shutting down unused instance", "instance", req.InstanceName)
		_ = sc.Close()
		delete(d.scrapers, req.InstanceName)
	}
}

// ApplyConfig updates all running scrapers.
func (d *distributor) ApplyConfig(cfg Config) error {
	d.scraperMut.Lock()
	defer d.scraperMut.Unlock()

	if d.closed.Load() {
		return fmt.Errorf("distributor shutting down")
	}

	var firstError error

	d.cfg = cfg
	for k, sc := range d.scrapers {
		err := sc.ApplyConfig(cfg)
		if err != nil {
			level.Error(d.log).Log("msg", "failed to update scraper", "instance", k, "err", err)
		}
		if firstError == nil {
			firstError = err
		}
	}

	return firstError
}

// ApplyConfig updates all running scrapers.
func (d *distributor) Close() error {
	d.scraperMut.Lock()
	defer d.scraperMut.Unlock()

	d.closed.Store(true)

	for _, sc := range d.scrapers {
		_ = sc.Close()
	}
	d.scrapers = make(map[string]*scraper)

	return nil
}
