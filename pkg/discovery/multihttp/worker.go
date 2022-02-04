package multihttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/agent/pkg/build"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

var (
	userAgent        = fmt.Sprintf("GrafanaAgent/%s", build.Version)
	matchContentType = regexp.MustCompile(`^(?i:application\/json(;\s*charset=("utf-8"|utf-8))?)$`)
)

// worker collects targets in the background.
type worker interface {
	// Run starts the worker and returns once ctx is canceled. Run must not be
	// called more than once.
	//
	// refresh should be called whenever there is a new set of targets that can
	// be retrieved.
	Run(ctx context.Context, refresh func())

	// Targets returns the latest set of targets the worker collected.
	Targets() []*targetgroup.Group
}

// httpWorker collects targets by invoking an HTTP endpoint.
type httpWorker struct {
	log log.Logger

	client          *http.Client
	url             string
	refreshInterval time.Duration
	refreshTimeout  time.Duration
	lastGroupsCount int

	mut            sync.RWMutex
	currentTargets []*targetgroup.Group
}

// newHTTPWorker initializes a new worker. It should be started by calling Run.
func newHTTPWorker(log log.Logger, client *http.Client, url string, cfg *Config) *httpWorker {
	return &httpWorker{
		log:             log,
		client:          client,
		url:             url,
		refreshInterval: time.Duration(cfg.RefreshInterval),
		refreshTimeout:  time.Duration(cfg.RefreshTimeout),
	}
}

// Run starts the worker and blocks until it is stopped. Run cannot be called
// more than once.
//
// refresh will be written to when new targets are available. Call Targets to
// retrieve them.
func (w *httpWorker) Run(ctx context.Context, refresh func()) {
	t := time.NewTicker(w.refreshInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			res, err := w.refresh(ctx)
			if err != nil {
				level.Error(w.log).Log("msg", "failed to refresh target", "err", err)
				// TODO(rfratto): metric?
				continue
			}

			// Only save the targets if the refresh succeeded. This prevents
			// intermittent failures from causing all targets to go away.
			w.mut.Lock()
			w.currentTargets = res
			w.mut.Unlock()

			refresh()
		case <-ctx.Done():
			return
		}
	}
}

func (w *httpWorker) Targets() []*targetgroup.Group {
	w.mut.RLock()
	defer w.mut.RUnlock()
	return w.currentTargets
}

// refresh refreshes the service.
func (w *httpWorker) refresh(ctx context.Context) ([]*targetgroup.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, w.refreshTimeout)
	defer cancel()

	req, err := http.NewRequest("GET", w.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Prometheus-Refresh-Interval-Seconds", strconv.FormatFloat(w.refreshInterval.Seconds(), 'f', -1, 64))

	resp, err := w.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("server returned HTTP status %s", resp.Status)
	} else if !matchContentType.MatchString(strings.TrimSpace(resp.Header.Get("Content-Type"))) {
		return nil, errors.Errorf("unsupported content type %q", resp.Header.Get("Content-Type"))
	}

	var targetGroups []*targetgroup.Group
	if err := json.NewDecoder(resp.Body).Decode(&targetGroups); err != nil {
		return nil, err
	}
	for i, group := range targetGroups {
		if group == nil {
			return nil, fmt.Errorf("null target group item found")
		}

		group.Source = urlSource(w.url, i)
		if group.Labels == nil {
			group.Labels = model.LabelSet{}
		}
		// TODO(rfratto): should any labels be injected here? Prometheus http_sd
		// injects __meta_url.
	}

	// Generate empty updates for sources that disappeared.
	l := len(targetGroups)
	for i := l; i < w.lastGroupsCount; i++ {
		targetGroups = append(targetGroups, &targetgroup.Group{Source: urlSource(w.url, i)})
	}
	w.lastGroupsCount = l

	return targetGroups, nil
}

// urlSource returns a source ID for the i-th target group per URL.
func urlSource(url string, i int) string {
	return fmt.Sprintf("%s:%d", url, i)
}
