package multihttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func Test_worker_refresh(t *testing.T) {
	emitGroups := []workerAPIResult{{
		Targets: []model.LabelSet{{
			"__address__": "localhost:123",
		}},
		Labels: model.LabelSet{
			"foo":  "bar",
			"fizz": "buzz",
		},
	}}

	t.Run("sets refresh interval and timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		w := newTestWorker(ctx, t, http.NotFoundHandler())

		require.Equal(t, time.Duration(DefaultConfig.RefreshInterval), w.refreshInterval)
		require.Equal(t, time.Duration(DefaultConfig.RefreshTimeout), w.refreshTimeout)
	})

	t.Run("sets expected headers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var gotHeaders http.Header
		h := func(rw http.ResponseWriter, req *http.Request) {
			gotHeaders = req.Header.Clone()

			rw.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(emitGroups)
		}

		w := newTestWorker(ctx, t, http.HandlerFunc(h))
		_, err := w.refresh(ctx)
		require.NoError(t, err)

		require.Equal(t, userAgent, gotHeaders.Get("User-Agent"))
		require.Equal(t, "application/json", gotHeaders.Get("Accept"))
		require.Equal(t, fmt.Sprintf("%d", int(w.refreshInterval.Seconds())), gotHeaders.Get("X-Prometheus-Refresh-Interval-Seconds"))
	})

	t.Run("generates target groups", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		h := func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(emitGroups)
		}

		w := newTestWorker(ctx, t, http.HandlerFunc(h))

		expect := []*targetgroup.Group{{
			Targets: []model.LabelSet{{
				"__address__": "localhost:123",
			}},
			Labels: model.LabelSet{
				"foo":  "bar",
				"fizz": "buzz",
			},
			Source: fmt.Sprintf("%s:0", w.url),
		}}

		actual, err := w.refresh(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, w.lastGroupsCount, "expected last groups count to be updated")
		require.Equal(t, expect, actual)
	})

	t.Run("generates empty target groups for disappeared groups", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		h := func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode([]workerAPIResult{})
		}

		w := newTestWorker(ctx, t, http.HandlerFunc(h))
		w.lastGroupsCount = 2

		expect := []*targetgroup.Group{
			{Source: fmt.Sprintf("%s:0", w.url)},
			{Source: fmt.Sprintf("%s:1", w.url)},
		}

		actual, err := w.refresh(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, w.lastGroupsCount, "expected last groups count to be updated")
		require.Equal(t, expect, actual)
	})

	t.Run("merges empty and non-empty groups", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		h := func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(emitGroups)
		}

		w := newTestWorker(ctx, t, http.HandlerFunc(h))
		w.lastGroupsCount = 2

		expect := []*targetgroup.Group{
			{
				Targets: []model.LabelSet{{
					"__address__": "localhost:123",
				}},
				Labels: model.LabelSet{
					"foo":  "bar",
					"fizz": "buzz",
				},
				Source: fmt.Sprintf("%s:0", w.url),
			},
			{
				// Targets and Labels should be unset here to signal there was a group
				// that went away.
				Source: fmt.Sprintf("%s:1", w.url),
			},
		}

		actual, err := w.refresh(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, w.lastGroupsCount, "expected last groups count to be updated")
		require.Equal(t, expect, actual)
	})
}

// newTestWorker creates a worker which will invoke the provided handler every
// refresh.
func newTestWorker(ctx context.Context, t *testing.T, h http.Handler) *httpWorker {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	lset := labels.FromMap(map[string]string{
		model.SchemeLabel:  "http",
		model.AddressLabel: srv.Listener.Addr().String(),
		APIPathLabel:       "/targets",
	})
	svc := NewService(lset, lset, nil)

	return newWorker(ctx, log.NewNopLogger(), http.DefaultClient, &DefaultConfig, svc)
}

type workerAPIResult targetgroup.Group

func (res workerAPIResult) MarshalJSON() ([]byte, error) {
	g := &struct {
		Targets []string       `json:"targets"`
		Labels  model.LabelSet `json:"labels,omitempty"`
	}{
		Targets: make([]string, 0, len(res.Targets)),
		Labels:  res.Labels,
	}
	for _, t := range res.Targets {
		g.Targets = append(g.Targets, string(t[model.AddressLabel]))
	}
	return json.Marshal(g)
}
