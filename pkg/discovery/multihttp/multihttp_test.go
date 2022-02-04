package multihttp

import (
	"context"
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
		return &fakeWorker{
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

type fakeWorker struct {
	onRunStart  func()
	onRunFinish func()
}

func (fw *fakeWorker) Run(ctx context.Context, refresh func()) {
	fw.onRunStart()
	defer fw.onRunFinish()
	<-ctx.Done()
}

func (fw *fakeWorker) Targets() []*targetgroup.Group { return nil }

func checkWorkers(t *testing.T, expect uint64, actual *atomic.Uint64, msgAndArgs ...interface{}) {
	t.Helper()

	require.Eventually(
		t, func() bool { return actual.Load() == expect },
		time.Second, 10*time.Millisecond,
		msgAndArgs...,
	)
}
