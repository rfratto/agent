// Package multihttp implements the multihttp service discovery: a service
// discovery which aggregates the result of multiple discovered services which
// expose individual http_sd endpoints.
//
// The Discovery mechanism is *not* automatically registered to Prometheus, and
// must be manually registered if you desire to use it directly as its own SD
// implementation.
package multihttp

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

// Discovery implements multihttp discovery.
//
// multihttp discovery embeds other Prometheus service discovery
// implementations to find services that expose http_sd endpoints.
//
// Discovered http_sd endpoints are then collected on an interval and their
// targets are aggregated into a flat set.
type Discovery struct {
	log log.Logger
	cfg *Config

	cli *http.Client
}

var _ discovery.Discoverer = (*Discovery)(nil)

// NewDiscovery returns a new multihttp Discovery.
func NewDiscovery(l log.Logger, c *Config) (*Discovery, error) {
	cli, err := config.NewClientFromConfig(c.HTTPClientConfig, "multihttp")
	if err != nil {
		return nil, err
	}

	return &Discovery{log: l, cfg: c, cli: cli}, nil
}

func (d *Discovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	var (
		key = "multihttp"
		dm  = discovery.NewManager(ctx, d.log, discovery.Name(key))

		pool         = newServicePool(d.log, d.cfg, d.cli)
		poolServices = make(chan []*targetgroup.Group)
	)

	err := dm.ApplyConfig(map[string]discovery.Configs{key: d.cfg.ServiceDiscoveryConfigs})
	if err != nil {
		level.Error(d.log).Log("msg", "failed to apply discoveries to multihttp", "err", err)
		return
	}

	// Run our SD manager and service pool in the background.
	go func() {
		_ = dm.Run()
		level.Debug(d.log).Log("msg", "multihttp discovery manager exited")
	}()
	go func() {
		pool.Run(ctx, poolServices)
		level.Debug(d.log).Log("msg", "multihttp service pool exited")
	}()

	for {
		select {
		case tsets := <-dm.SyncCh():
			// Our discovery manager found a new set of targets. Pass them over to
			// our pool for processing.
			if len(tsets) != 1 {
				level.Error(d.log).Log("msg", "unexpected number of target sets; expected 1", "len", len(tsets))
				continue
			} else if _, ok := tsets[key]; !ok {
				level.Error(d.log).Log("msg", "missing multihttp target set; ignoring targets")
				continue
			}

			// Request our pool to resync its services.
			poolServices <- tsets[key]

		case discovered := <-pool.TargetsCh():
			// Forward discovered targets to our parent
			up <- discovered

		case <-ctx.Done():
			return
		}
	}
}

type servicePool struct {
	log log.Logger
	cfg *Config
	cli *http.Client

	targets     chan []*targetgroup.Group
	triggerSend chan struct{} // Schedules the full set of targets to be recalculated across all the workers.

	// TODO(rfratto): expose allServices, droppedServices via API

	mut             sync.Mutex
	allServices     []*Service
	droppedServices []*Service
	activeWorkers   map[uint64]*worker // Running workers for services
}

func newServicePool(l log.Logger, cfg *Config, cli *http.Client) *servicePool {
	return &servicePool{
		log: l,
		cfg: cfg,
		cli: cli,

		targets:     make(chan []*targetgroup.Group),
		triggerSend: make(chan struct{}, 1),
	}
}

// TargetsCh returns the channel of targets the service pool discovered across
// all services.
func (p *servicePool) TargetsCh() <-chan []*targetgroup.Group { return p.targets }

// Run will read from services and schedule workers from the discovered targets
// until ctx is canceled.
func (p *servicePool) Run(ctx context.Context, services <-chan []*targetgroup.Group) {
	// Clear out running workers by scheduling an empty slice.
	defer func() {
		p.sync(ctx, nil)

		select {
		case p.targets <- nil:
		default:
		}
	}()

	checkUpdate := time.NewTicker(5 * time.Second)
	defer checkUpdate.Stop()

	for {
		select {
		case svcs := <-services:
			// There's a new set of services, manage loops for each of them.
			p.sync(ctx, svcs)

		case <-checkUpdate.C: // Throttle checking for updates
			select {
			case <-p.triggerSend:
				// TODO(rfratto): counter for # updates
				select {
				case p.targets <- p.allTargets():
					continue
				default:
				}

				level.Debug(p.log).Log("msg", "targets reader blocked, will try again later")
				select {
				case p.triggerSend <- struct{}{}:
				default:
					// trigger already scheduled
				}
			default:
				// No update triggered, wait for the next tick
				continue
			}

		case <-ctx.Done():
			return
		}
	}
}

// sync proceses a targets and schedules services to be watched.
func (p *servicePool) sync(ctx context.Context, groups []*targetgroup.Group) {
	var (
		allServices, droppedServices []*Service
	)

	for _, group := range groups {
		svcs, err := servicesFromGroup(group, p.cfg)
		if err != nil {
			level.Error(p.log).Log("msg", "failed to create services from targets", "err", err)
			// TODO(rfratto): error count metric?
		}
		for _, s := range svcs {
			if s.Labels().Len() > 0 {
				allServices = append(allServices, s)
			} else {
				droppedServices = append(droppedServices, s)
			}
		}
	}

	p.mut.Lock()
	defer p.mut.Unlock()

	p.droppedServices = droppedServices
	p.allServices = allServices
	p.schedule(ctx, allServices)

	// TODO(rfratto): metrics for sync interval length + syncs counter?
}

// schedule manages a set of workers from svcs. services are deduped based on
// their hashes of their labels and URL. Only call with mut held.
//
// workers for services that go away between calls will be removed, while new
// workers will be created for new services.
func (p *servicePool) schedule(ctx context.Context, svcs []*Service) {
	uniqueWorkers := make(map[uint64]*worker)

	for _, svc := range svcs {
		hash := svc.hash()

		if _, ok := p.activeWorkers[hash]; !ok {
			// This is a new service, we need to spin up a worker for it.
			w := newWorker(
				context.Background(),
				p.log,
				p.cli,
				p.cfg,
				svc,
			)

			p.activeWorkers[hash] = w
			uniqueWorkers[hash] = w
		} else {
			if _, ok := uniqueWorkers[hash]; !ok {
				// This is an existing or duplicated worker.
				uniqueWorkers[hash] = nil
			}
			// The discovered labels may have changed for an existing worker, so we
			// want to copy them over.
			p.activeWorkers[hash].svc.setDiscoveredLabels(svc.DiscoveredLabels())
		}
	}

	var stoppedWorkers sync.WaitGroup

	for hash, w := range p.activeWorkers {
		if _, ok := uniqueWorkers[hash]; !ok {
			stoppedWorkers.Add(1)
			go func(w *worker) {
				defer stoppedWorkers.Done()
				w.Stop()
			}(w)
			delete(p.activeWorkers, hash)
		}
	}

	// TODO(rfratto): code from Prometheus:
	// targetScrapePoolTargetsAdded.WithLabelValues(sp.config.JobName).Set(float64(len(uniqueLoops)))

	for _, w := range uniqueWorkers {
		if w != nil {
			go w.Run(p.triggerSend)
		}
	}

	// Wait for stopped workers to terminate.
	stoppedWorkers.Wait()

	// Immediately try to schedule a refresh to handle the changed workers.
	select {
	case p.triggerSend <- struct{}{}:
	default:
	}
}

// allTargets calculcates the combined set of targets across all workers.
func (p *servicePool) allTargets() []*targetgroup.Group {
	// TODO(rfratto): dedupe?

	p.mut.Lock()
	defer p.mut.Unlock()

	var res []*targetgroup.Group
	for _, w := range p.activeWorkers {
		// TODO(rfratto): mutate source to be multihttp
		res = append(res, w.Targets()...)
	}
	return res
}
