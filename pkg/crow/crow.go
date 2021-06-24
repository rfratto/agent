// Package crow implements a correctness checker tool similar to Loki Canary.
// Inspired by Cortex test-exporter.
package crow

import (
	"flag"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	totalScrapes prometheus.Counter
	totalSamples prometheus.Counter
	totalResults *prometheus.CounterVec
	pendingSets  prometheus.Gauge

	cachedCollectors []prometheus.Collector
}

func newMetrics() *metrics {
	var m metrics

	m.totalScrapes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "crow_test_scrapes_total",
		Help: "Total number of generated test sample sets",
	})

	m.totalSamples = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "crow_test_samples_total",
		Help: "Total number of generated test samples",
	})

	m.totalResults = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "crow_test_sample_results_total",
		Help: "Total validation results of test samples",
	}, []string{"result"}) // result is either "success", "missing", or "mismatch"

	m.pendingSets = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "crow_test_pending_validations",
		Help: "Total number of pending validations to perform",
	})

	return &m
}

func (m *metrics) collectors() []prometheus.Collector {
	if m.cachedCollectors == nil {
		m.cachedCollectors = []prometheus.Collector{
			m.totalScrapes,
			m.totalSamples,
			m.totalResults,
			m.pendingSets,
		}
	}
	return m.cachedCollectors
}

func (m *metrics) Describe(ch chan<- *prometheus.Desc) {
	for _, c := range m.collectors() {
		c.Describe(ch)
	}
}

func (m *metrics) Collect(ch chan<- prometheus.Metric) {
	for _, c := range m.collectors() {
		c.Collect(ch)
	}
}

// Config for the Crow metrics checker.
type Config struct {
	// Base URL of the Prometheus server.
	PrometheusAddr string
	// UserID to use when querying.
	UserID string
	// Extra seletors to be included in queries. i.e., cluster="prod"
	ExtraSelectors string
	// Maximum amount of times a validation can be attempted.
	MaximumValidations int
}

// RegisterFlags registers flags for the config to the given FlagSet.
func (c *Config) RegisterFlags(f *flag.FlagSet) {
	f.StringVar(&c.PrometheusAddr, "prometheus-addr", DefaultConfig.PrometheusAddr, "Root URL of the Prometheus API to query against")
	f.StringVar(&c.UserID, "user-id", DefaultConfig.UserID, "UserID to attach to query. Useful for querying multi-tenated Cortex.")
	f.StringVar(&c.ExtraSelectors, "extra-selectors", DefaultConfig.ExtraSelectors, "Extra selectors to include in queries, useful for identifying different instances of this job.")
	f.IntVar(&c.MaximumValidations, "maximum-validations", DefaultConfig.MaximumValidations, "Maximum number of times to try validating a sample")
}

// DefaultConfig holds defaults for Crow settings.
var DefaultConfig = Config{
	MaximumValidations: 5,
}

// Crow is a collectness checker that validates scraped metrics reach a
// Prometheus-compatible server with the same values and roughly the same
// timestamp.
//
// Crow exposes two sets of metrics:
//
// 1. Test metrics, where each scrape generates a validation job.
// 2. State metrics, exposing state of the Crow checker itself.
//
// These two metrics should be exposed via different endpoints, and only state
// metrics are safe to be manually collecetd from.
//
// Collecting from the set of test metrics generates a validation job, where
// Crow will query the Prometheus API to ensure the metrics that were scraped
// were written with (approximately) the same timestamp as the scrape time
// and with (approximately) the same floatnig point values exposed in the
// scrape.
//
// If a set of test metrics were not found and retries have been exhausted,
// or if the metrics were found but the values did not match, the error
// counter will increase.
type Crow struct {
	cfg Config
	m   *metrics

	wg   sync.WaitGroup
	quit chan struct{}

	pendingMtx sync.Mutex
	pending    []*sample
}

// New creates a new Crow.
func New(cfg Config) (*Crow, error) {
	c := &Crow{
		cfg: cfg,
		m:   newMetrics(),
	}

	c.wg.Add(1)
	go c.runLoop()
	return c, nil
}

func (c *Crow) runLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.quit:
			return
		case <-ticker.C:
			c.checkPending()
		}
	}
}

// checkPending iterates over all pending samples. Samples that are ready
// are immediately validated. Samples are requeued if they're not ready or
// not found during validation.
func (c *Crow) checkPending() {
	c.pendingMtx.Lock()
	defer c.pendingMtx.Unlock()

	now := time.Now()

	requeued := []*sample{}
	for _, s := range c.pending {
		if !s.Ready(now) {
			requeued = append(requeued, s)
			continue
		}

		requeue := c.validate(s)
		if requeue {
			requeued = append(requeued, s)
		}
	}
	c.pending = requeued
}

// validate validates a sample. If the sample should be requeued (i.e.,
// couldn't be found), returns true.
func (c *Crow) validate(b *sample) (requeue bool) {
	// TODO(rfratto): do validation

	// TODO(rfratto): only do this if the validation failed
	b.ValidationAttempt++
	if b.ValidationAttempt >= c.cfg.MaximumValidations {
		return false
	}
	return true
}

// TestMetrics exposes a collector of test metrics. Each collection will
// schedule a validation job.
func (c *Crow) TestMetrics() prometheus.Collector {
	panic("NYI")
}

// StateMetrics exposes metrics of Crow itself. These metrics are not validated
// for presence in the remote system.
func (c *Crow) StateMetrics() prometheus.Collector { return c.m }

// Stop stops crow. Panics if Stop is called more than once.
func (c *Crow) Stop() {
	close(c.quit)
	c.wg.Wait()
}

type sample struct {
	ScrapeTime time.Time
	Labels     prometheus.Labels
	Value      float64

	// How many times this sample has attempted to be valdated. Starts at 0.
	ValidationAttempt int
}

// Ready checks if this sample is ready to be validated.
func (s *sample) Ready(now time.Time) bool {
	// Exponential backoff from 500ms up (500ms * 2^attempt).
	backoff := (500 * time.Millisecond) * time.Duration(math.Pow(2, float64(s.ValidationAttempt)))
	return now.After(s.ScrapeTime.Add(backoff))
}
