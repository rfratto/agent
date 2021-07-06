package metrics

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/agent/pkg/cluster"
	"github.com/grafana/agent/pkg/metrics/internal/metricspb"
	"github.com/grafana/agent/pkg/prom"
	"github.com/grafana/agent/pkg/prom/instance"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/config"
	"github.com/rfratto/croissant/id"
	"github.com/rfratto/croissant/node"
	"google.golang.org/grpc"
)

// Config defines the configuration for the entire set of Prometheus client instances,
// along with a global configuration.
type Config struct {
	Global  instance.GlobalConfig `yaml:"global,omitempty"`
	Configs []instance.Config     `yaml:"configs,omitempty,omitempty"`

	// Flag only:
	WALDir           string        `yaml:"-"`
	WALCleanupAge    time.Duration `yaml:"-"`
	WALCleanupPeriod time.Duration `yaml:"-"`

	// Deprecated:
	InstanceMode           instance.Mode `yaml:"-,omitempty"`
	InstanceRestartBackoff time.Duration `yaml:"-"`
}

// DefaultConfig is the default settings for the Prometheus-lite client.
var DefaultConfig = Config{
	Global:                 instance.DefaultGlobalConfig,
	InstanceRestartBackoff: instance.DefaultBasicManagerConfig.InstanceRestartBackoff,
	WALCleanupAge:          prom.DefaultCleanupAge,
	WALCleanupPeriod:       prom.DefaultCleanupPeriod,
	InstanceMode:           instance.ModeDistinct,
}

// RegisterFlags defines flags corresponding to the Config.
func (c *Config) RegisterFlags(f *flag.FlagSet) {
	f.StringVar(&c.WALDir, "metrics.data-dir", "", "storage for metrics WAL")
	f.DurationVar(&c.WALCleanupAge, "metrics.data-cleanup-age", DefaultConfig.WALCleanupAge, "remove unused WALs older than this")
	f.DurationVar(&c.WALCleanupPeriod, "metrics.data-cleanup-period", DefaultConfig.WALCleanupPeriod, "how often to check for unused data")
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultConfig
	type plain Config
	return unmarshal((*plain)(c))
}

// ApplyDefaults applies default values to the Config and validates it.
func (c *Config) ApplyDefaults() error {
	needWAL := len(c.Configs) > 0
	if needWAL && c.WALDir == "" {
		return errors.New("no wal_directory configured")
	}

	usedNames := map[string]struct{}{}

	for i := range c.Configs {
		name := c.Configs[i].Name
		if err := c.Configs[i].ApplyDefaults(c.Global); err != nil {
			// Try to show a helpful name in the error
			if name == "" {
				name = fmt.Sprintf("at index %d", i)
			}

			return fmt.Errorf("error validating instance %s: %w", name, err)
		}

		if _, ok := usedNames[name]; ok {
			return fmt.Errorf(
				"prometheus instance names must be unique. found multiple instances with name %s",
				name,
			)
		}
		usedNames[name] = struct{}{}
	}

	return nil
}

// Metrics collects and delivers Prometheus metrics.
type Metrics struct {
	log log.Logger
	cfg Config

	cluster     *cluster.Cluster
	discoverers map[string]*discoveryJob
	distributor *distributor

	reload chan struct{}

	stop, done chan struct{}
}

// New creates a new clustered Metrics collector. Targets will be distributed
// roughly evenly throughout the cluster.
func New(reg prometheus.Registerer, logger log.Logger, cfg Config, cluster *cluster.Cluster) (*Metrics, error) {
	dist, err := newDistributor(reg, logger, cfg, cluster)
	if err != nil {
		return nil, err
	}

	m := &Metrics{
		log: log.With(logger, "component", "metrics"),
		cfg: cfg,

		cluster:     cluster,
		discoverers: make(map[string]*discoveryJob),
		distributor: dist,

		reload: make(chan struct{}, 5),

		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	if err := m.ApplyConfig(cfg); err != nil {
		return nil, err
	}

	// TODO(rfratto): Can't create a WAL cleaner yet because no concept of an instance manager here.

	go m.run()
	return m, nil
}

func (m *Metrics) run() {
	defer close(m.done)

	level.Debug(m.log).Log("msg", "waiting to join cluster before starting jobs")
	m.cluster.WaitJoined()
	level.Debug(m.log).Log("msg", "joined cluster, starting loop")

Loop:
	for {
		select {
		case <-m.stop:
			m.distributor.Close()
			for _, d := range m.discoverers {
				d.Close()
			}
			break Loop
		case <-m.reload:
			m.syncDiscoverers()
			m.distributor.Reshard(context.Background())
		}
	}
}

// syncDiscoverers manages Prometheus SD components.
func (m *Metrics) syncDiscoverers() {
	level.Info(m.log).Log("msg", "syncing discoverers")
	defer level.Info(m.log).Log("msg", "done syncing discoverers")

	newJobs := map[string]struct{}{}

	for _, ic := range m.cfg.Configs {
		ourJobs := make([]*config.ScrapeConfig, 0, len(ic.ScrapeConfigs))
		jobNames := make([]string, 0, len(ic.ScrapeConfigs))

		for _, sc := range ic.ScrapeConfigs {
			_, self, err := m.cluster.Node().NextPeer(getKey(ic.Name + "/" + sc.JobName))
			if err != nil {
				// We really don't want to leave a job unscraped. If there's a bug in
				// routing, we'll force ownership of a job. If multiple nodes do this,
				// there will be out of order / duplicate samples, but at least the
				// metrics will be sent.
				level.Warn(m.log).Log("msg", "could not determine ownership for scrape job, forcing ownership", "err", err)
				ourJobs = append(ourJobs, sc)
			} else if self {
				ourJobs = append(ourJobs, sc)
				jobNames = append(jobNames, sc.JobName)
			}
		}

		level.Debug(m.log).Log("msg", "determined responsibility for scrape job", "instance", ic.Name, "jobs", strings.Join(jobNames, ","))

		if len(ourJobs) == 0 {
			continue
		}

		disc, ok := m.discoverers[ic.Name]

		var err error
		if !ok {
			disc, err = newDiscoveryJob(m.log, ic.Name, m.distributor, ourJobs)
			m.discoverers[ic.Name] = disc
		} else {
			err = disc.ApplyConfig(ourJobs)
		}
		if err != nil {
			level.Error(m.log).Log("msg", "failed to send jobs to discoverer", "instance", ic.Name, "err", err)
		}

		// Update or start new SD for ic w/ ourJobs
		newJobs[ic.Name] = struct{}{}
	}

	// Delete jobs that have gone away or where we lost ownership.
	for job := range m.discoverers {
		_, ok := newJobs[job]
		if !ok {
			level.Debug(m.log).Log("msg", "shutting down unused discoverer", "job", job)
			_ = m.discoverers[job].Close()
			delete(m.discoverers, job)
		}
	}
}

func (m *Metrics) PeersChanged(ps []node.Peer) {
	level.Info(m.log).Log("msg", "detected peers changed")
	m.reload <- struct{}{}
}

func (m *Metrics) ApplyConfig(cfg Config) error {
	m.cfg = cfg
	m.reload <- struct{}{}
	return nil
}

func (m *Metrics) WireGRPC(s *grpc.Server) {
	metricspb.RegisterScraperServer(s, m.distributor)
}

func (m *Metrics) Close() error {
	close(m.stop)
	<-m.done
	return nil
}

func getKey(s string) id.ID {
	return cluster.NewKeyGenerator(32).Get(s)
}
