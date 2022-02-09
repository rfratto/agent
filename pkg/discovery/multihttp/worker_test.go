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
	"github.com/stretchr/testify/require"
)

// Mock group to use for testing
var sampleGroup = targetgroup.Group{
	Targets: []model.LabelSet{{
		"__address__": "localhost:123",
	}},
	Labels: model.LabelSet{
		"foo":  "bar",
		"fizz": "buzz",
	},
}

func Test_worker(t *testing.T) {
	t.Run("sets refresh interval and timeout", func(t *testing.T) {
		w := newTestWorker(t, http.NotFoundHandler().ServeHTTP)

		require.Equal(t, time.Duration(DefaultConfig.RefreshInterval), w.refreshInterval)
		require.Equal(t, time.Duration(DefaultConfig.RefreshTimeout), w.refreshTimeout)
	})
}

func Test_worker_Run(t *testing.T) {
	t.Run("invokes callback after refreshing", func(t *testing.T) {
		h := func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode([]workerAPIGroup{})
		}

		var (
			refreshInterval   = 10 * time.Millisecond
			testDuration      = refreshInterval * 5
			expectedRefreshes = (testDuration / refreshInterval) - 1 // Give a buffer of missing 1 refresh
		)

		w := newTestWorker(t, http.HandlerFunc(h))
		w.refreshInterval = refreshInterval

		refreshCh := make(chan struct{})
		defer func() {
			for range refreshCh {
				// Drain the channel
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), testDuration)
		defer cancel()

		go func() {
			defer close(refreshCh)
			w.Run(ctx, func() {
				refreshCh <- struct{}{}
			})
		}()

		// Wait for all of our refreshes to trigger
		for i := 0; i < int(expectedRefreshes); i++ {
			select {
			case <-refreshCh:
				// No-op
			case <-ctx.Done():
				require.FailNow(t, "missing refresh notification", "refresh number %d missing", i)
			}
		}
	})
}

func Test_worker_refresh(t *testing.T) {
	tt := []struct {
		name  string
		mut   func(*httpWorker) // Use to mutate http worker settings. Optional.
		input []*targetgroup.Group

		// Validate is used to validate the result of a request. The worker,
		// http.Request, and resulting targets are provided.
		validate func(t *testing.T, w *httpWorker, req *http.Request, result []*targetgroup.Group)
	}{
		{
			name:  "sets expected headers",
			input: nil,

			validate: func(t *testing.T, w *httpWorker, req *http.Request, _ []*targetgroup.Group) {
				require.Equal(t, userAgent, req.Header.Get("User-Agent"))
				require.Equal(t, "application/json", req.Header.Get("Accept"))
				require.Equal(t, fmt.Sprintf("%d", int(w.refreshInterval.Seconds())), req.Header.Get("X-Prometheus-Refresh-Interval-Seconds"))
			},
		},
		{
			name:  "generates target groups",
			input: []*targetgroup.Group{&sampleGroup},

			validate: func(t *testing.T, w *httpWorker, _ *http.Request, result []*targetgroup.Group) {
				expect := []*targetgroup.Group{
					groupWithSource(&sampleGroup, urlSource(w.url, 0)),
				}
				require.Equal(t, expect, result)
			},
		},
		{
			name:  "updates last group count",
			input: []*targetgroup.Group{&sampleGroup},

			validate: func(t *testing.T, w *httpWorker, _ *http.Request, _ []*targetgroup.Group) {
				require.Equal(t, 1, w.lastGroupsCount, "expected last groups count to be updated")
			},
		},
		{
			name:  "generate empty target groups for removed groups",
			input: nil,
			mut:   func(hw *httpWorker) { hw.lastGroupsCount = 2 },

			validate: func(t *testing.T, w *httpWorker, _ *http.Request, result []*targetgroup.Group) {
				expect := []*targetgroup.Group{
					{Source: urlSource(w.url, 0)},
					{Source: urlSource(w.url, 1)},
				}

				require.Equal(t, 0, w.lastGroupsCount, "expected last groups count to be updated")
				require.Equal(t, expect, result)
			},
		},
		{
			name:  "merges empty and non-empty groups",
			input: []*targetgroup.Group{&sampleGroup},
			mut:   func(hw *httpWorker) { hw.lastGroupsCount = 2 },

			validate: func(t *testing.T, w *httpWorker, _ *http.Request, result []*targetgroup.Group) {
				expect := []*targetgroup.Group{
					groupWithSource(&sampleGroup, urlSource(w.url, 0)),
					{Source: urlSource(w.url, 1)},
				}

				require.Equal(t, 1, w.lastGroupsCount, "expected last groups count to be updated")
				require.Equal(t, expect, result)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var gotReq *http.Request

			w := newTestWorker(t, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				gotReq = r.Clone(r.Context())

				rw.Header().Add("Content-Type", "application/json")
				_ = json.NewEncoder(rw).Encode(workerAPIResult(tc.input))
			}))
			if tc.mut != nil {
				tc.mut(w)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			actual, err := w.refresh(ctx)
			require.NoError(t, err)

			if tc.validate != nil {
				tc.validate(t, w, gotReq, actual)
			}
		})
	}
}

func Test_worker_Targets(t *testing.T) {
	emitGroups := []*targetgroup.Group{&sampleGroup}

	t.Run("returns targets from latest refresh", func(t *testing.T) {
		h := func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(workerAPIResult(emitGroups))
		}

		w := newTestWorker(t, http.HandlerFunc(h))
		w.refreshInterval = 10 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		w.Run(ctx, cancel)
		require.ErrorIs(t, ctx.Err(), context.Canceled)

		expect := []*targetgroup.Group{
			groupWithSource(&sampleGroup, fmt.Sprintf("%s:0", w.url)),
		}
		require.Equal(t, expect, w.Targets())
	})
}

// newTestWorker creates a worker which will invoke the provided handler every
// refresh.
func newTestWorker(t *testing.T, h http.HandlerFunc) *httpWorker {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	serviceUrl := fmt.Sprintf("http://%s/targets", srv.Listener.Addr())
	return newHTTPWorker(log.NewNopLogger(), http.DefaultClient, serviceUrl, &DefaultConfig)
}

type workerAPIGroup targetgroup.Group

func (res workerAPIGroup) MarshalJSON() ([]byte, error) {
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

type workerAPIResult []*targetgroup.Group

func (res workerAPIResult) MarshalJSON() ([]byte, error) {
	apiGroups := make([]*workerAPIGroup, len(res))
	for i, group := range res {
		apiGroups[i] = (*workerAPIGroup)(group)
	}
	return json.Marshal(apiGroups)
}
