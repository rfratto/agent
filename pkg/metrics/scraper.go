package metrics

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/agent/pkg/metrics/internal/metricspb"
	"github.com/grafana/agent/pkg/prom/instance"
	"github.com/grafana/agent/pkg/prom/wal"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/storage/remote"
)

// scraper creates a Prometheus metrics scraper, WAL, and remote write. It handles
// scrape jobs for an entire metrics instance key.
type scraper struct {
	log log.Logger

	// Instance name to pull config from.
	name string

	tracker *targetTracker

	cancel context.CancelFunc // Stops the scraper.
	done   chan struct{}      // Closed when scraper exits.

	storage storage.Storage
	remote  *remote.Storage
	wal     *wal.Storage
	sm      *scrape.Manager
}

func newScraper(logger log.Logger, reg prometheus.Registerer, cfg Config, name string) (*scraper, error) {
	logger = log.With(logger, "instance", name)
	var (
		walLogger    = log.With(logger, "prometheus_component", "wal")
		remoteLogger = log.With(logger, "prometheus_component", "remote")
		scrapeLogger = log.With(logger, "prometheus_component", "scrape")
	)

	// Create Prometheus storage (WAL & remote write)
	walDir := filepath.Join(cfg.WALDir, name)
	w, err := wal.NewStorage(walLogger, reg, walDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAL at %s: %w", walDir, err)
	}

	// TODO(rfratto): make RemoteFlushDeadline a global setting (time.Minute below)
	rsm := &readyScrapeManager{}
	rs := remote.NewStorage(remoteLogger, reg, w.StartTime, walDir, time.Minute, rsm)
	s := storage.NewFanout(logger, w, rs)

	// Create Prometheus metrics scraper
	sm := scrape.NewManager(scrapeLogger, s)

	ctx, cancel := context.WithCancel(context.Background())
	res := &scraper{
		name: name,
		log:  logger,

		tracker: newTargetTracker(),

		cancel: cancel,
		done:   make(chan struct{}),

		storage: s,
		remote:  rs,
		wal:     w,
		sm:      sm,
	}

	if err := res.ApplyConfig(cfg); err != nil {
		return nil, err
	}
	rsm.Set(sm)

	go res.run(ctx)
	return res, nil
}

// ApplyConfig updates the config of the scraper.
func (s *scraper) ApplyConfig(cfg Config) error {
	var instConfig instance.Config
	for _, inst := range cfg.Configs {
		if inst.Name == s.name {
			instConfig = inst
			break
		}
	}
	if instConfig.Name == "" {
		return fmt.Errorf("local config does not contain instance %s: will not scrape", s.name)
	}

	err := s.remote.ApplyConfig(&config.Config{
		GlobalConfig:       cfg.Global.Prometheus,
		RemoteWriteConfigs: instConfig.RemoteWrite,
	})
	if err != nil {
		return fmt.Errorf("failed to apply instance config %s to remote storage: %w", s.name, err)
	}

	err = s.sm.ApplyConfig(&config.Config{
		GlobalConfig:  cfg.Global.Prometheus,
		ScrapeConfigs: instConfig.ScrapeConfigs,
	})
	if err != nil {
		return fmt.Errorf("failed to apply instance config %s to scraper: %w", s.name, err)
	}

	return nil
}

// Track will track the targets from req. Returns the total number of targets
// being tracked.
func (s *scraper) Track(ctx context.Context, req *metricspb.ScrapeTargetsRequest) (totalTargets int, err error) {
	return s.tracker.Track(ctx, req)
}

// Close closes the scraper.
func (s *scraper) Close() error {
	s.cancel()
	<-s.done
	return nil
}

func (s *scraper) run(ctx context.Context) {
	defer close(s.done)
	defer level.Debug(s.log).Log("msg", "scraper exiting")

	var rg run.Group

	// Wait for ctx to complete
	{
		rg.Add(func() error {
			<-ctx.Done()
			return nil
		}, func(e error) {})
	}

	{
		// TODO(rfratto): truncation loop
	}

	// Scrape manager
	{
		rg.Add(
			func() error {
				err := s.sm.Run(s.tracker.SyncCh())
				level.Debug(s.log).Log("msg", "scrape manager stopped")
				return err
			},
			func(e error) {
				level.Debug(s.log).Log("msg", "stopping scrape manager...")
				s.sm.Stop()

				// TODO(rfratto): write stale on shutdown?

				level.Debug(s.log).Log("msg", "closing storage...")
				s.storage.Close()
			},
		)
	}

	level.Debug(s.log).Log("msg", "running scraper")
	err := rg.Run()
	if err != nil {
		level.Error(s.log).Log("msg", "scraper exited with error", "err", err)
	}
}

// ErrNotReady is returned when the scrape manager is used but has not been
// initialized yet.
var ErrNotReady = errors.New("Scrape manager not ready")

// readyScrapeManager allows a scrape manager to be retrieved. Even if it's set at a later point in time.
type readyScrapeManager struct {
	mtx sync.RWMutex
	m   *scrape.Manager
}

// Set the scrape manager.
func (rm *readyScrapeManager) Set(m *scrape.Manager) {
	rm.mtx.Lock()
	defer rm.mtx.Unlock()

	rm.m = m
}

// Get the scrape manager. If is not ready, return an error.
func (rm *readyScrapeManager) Get() (*scrape.Manager, error) {
	rm.mtx.RLock()
	defer rm.mtx.RUnlock()

	if rm.m != nil {
		return rm.m, nil
	}

	return nil, ErrNotReady
}
