package multihttp

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/agent/pkg/util"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func Test_servicePool_schedule(t *testing.T) {
	running := atomic.NewUint64(0)

	sp := newServicePool(util.TestLogger(t), &DefaultConfig, http.DefaultClient)
	sp.newWorker = func(opts workerOptions) worker {
		return &notifyWorker{
			onRunStart:  func() { running.Inc() },
			onRunFinish: func() { running.Dec() },
		}
	}

	// Apply new workers
	sp.schedule(context.Background(), []*Service{
		NewService(labels.FromStrings(model.AddressLabel, "service-a"), nil, nil),
		NewService(labels.FromStrings(model.AddressLabel, "service-b"), nil, nil),
	})
	checkWorkers(t, 2, running, "expected new workers to be scheduled")

	// Re-apply the same set of services
	sp.schedule(context.Background(), []*Service{
		NewService(labels.FromStrings(model.AddressLabel, "service-a"), nil, nil),
		NewService(labels.FromStrings(model.AddressLabel, "service-b"), nil, nil),
	})
	checkWorkers(t, 2, running, "expected the same number of workers to be scheduled")

	// Remove a service
	sp.schedule(context.Background(), []*Service{
		NewService(labels.FromStrings(model.AddressLabel, "service-b"), nil, nil),
	})
	checkWorkers(t, 1, running, "expected a worker to be removed")
}

func Test_servicePool_allTargets(t *testing.T) {
	var workerCount atomic.Uint64

	sp := newServicePool(util.TestLogger(t), &DefaultConfig, http.DefaultClient)
	sp.newWorker = func(_ workerOptions) worker {
		num := workerCount.Inc()
		return &fakeWorker{
			targets: []*targetgroup.Group{
				groupWithSource(&sampleGroup, fmt.Sprintf("worker-%d", num)),
			},
		}
	}

	// Apply new workers
	sp.schedule(context.Background(), []*Service{
		NewService(labels.FromStrings(model.AddressLabel, "service-a"), nil, nil),
		NewService(labels.FromStrings(model.AddressLabel, "service-b"), nil, nil),
	})

	// Check that all targets combines our targets from both workers
	expect := []*targetgroup.Group{
		// We expect the source to be overwritten to "multihttp" here. This is
		// required to have Prometheus treat the set of targets here as the
		// definitive target set from SD.
		groupWithSource(&sampleGroup, "multihttp"),
		groupWithSource(&sampleGroup, "multihttp"),
	}
	require.Equal(t, expect, sp.allTargets())
}

type notifyWorker struct {
	onRunStart  func()
	onRunFinish func()
}

func (nw *notifyWorker) Run(ctx context.Context, refresh func()) {
	nw.onRunStart()
	defer nw.onRunFinish()
	<-ctx.Done()
}

func (nw *notifyWorker) Targets() []*targetgroup.Group { return nil }

func checkWorkers(t *testing.T, expect uint64, actual *atomic.Uint64, msgAndArgs ...interface{}) {
	t.Helper()

	require.Eventually(
		t, func() bool { return actual.Load() == expect },
		time.Second, 10*time.Millisecond,
		msgAndArgs...,
	)
}

type fakeWorker struct{ targets []*targetgroup.Group }

func (fw *fakeWorker) Run(ctx context.Context, refresh func()) { <-ctx.Done() }
func (fw *fakeWorker) Targets() []*targetgroup.Group           { return fw.targets }
