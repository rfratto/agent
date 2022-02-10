package discoveryext

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

// Watcher stores the latest set of targets from a Discoverer. Watcher implements
// http.Handler which writes the current set of targets using the http_sd API.
type Watcher struct {
	cancel context.CancelFunc
	exited chan struct{}

	mut sync.RWMutex
	// Some Discoverers(e.g. k8s) send only the updates for a given target group,
	// so cache entries are keyed by targetgroup.Group.Source.
	entries map[string]*targetgroup.Group
}

// NewWatcher creates a new Watcher. Watcher watches for targets in the
// background until the Watcher is Closed.
func NewWatcher(disc discovery.Discoverer) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Watcher{
		cancel:  cancel,
		exited:  make(chan struct{}),
		entries: make(map[string]*targetgroup.Group),
	}

	go c.run(ctx, disc)
	return c
}

func (c *Watcher) run(ctx context.Context, disc discovery.Discoverer) {
	defer close(c.exited)

	up := make(chan []*targetgroup.Group)
	go disc.Run(ctx, up)

	for {
		select {
		case <-ctx.Done():
			return
		case groups := <-up:
			c.updateCache(groups)
		}
	}
}

func (c *Watcher) updateCache(groups []*targetgroup.Group) {
	c.mut.Lock()
	defer c.mut.Unlock()

	// This is similar to discovery.Manager.updateGroup from Prometheus: Some
	// discoverers send nil groups or partial groups, so we need to merge it into
	// the final set.
	//
	// Note that there is no way for entries from a source to be removed from the
	// map. This also mirrors the Prometheus behavior.
	for _, group := range groups {
		if group != nil {
			c.entries[group.Source] = group
		}
	}
}

// Entries returns the current cached target groups. The returned pointers
// should not be modified.
func (c *Watcher) Entries() []*targetgroup.Group {
	c.mut.RLock()
	defer c.mut.RUnlock()

	res := make([]*targetgroup.Group, 0, len(c.entries))
	for _, group := range c.entries {
		res = append(res, group)
	}
	return res
}

// Close stops the Watcher and returns after it's fully stopped.
func (c *Watcher) Close() error {
	c.cancel()
	<-c.exited
	return nil
}

// ServeHTTP implements http.Handler. The current cached target groups will be
// written in the http_sd API format.
func (c *Watcher) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// ServeHTTP writes in the http_sd API format, which forces us to convert the
	// raw targets into a compatible representation.
	//
	// Targets from a group only propagate their address label, so we split each
	// target into its own group and give each group all the non-address labels
	// from the target.
	var (
		ents    = c.Entries()
		apiEnts = make([]*TargetGroup, 0, len(ents))
	)

	// TODO(rfratto): hash the resulting labels and use to dedupe groups that are
	// the same except for the address.
	for _, group := range ents {
		for _, target := range group.Targets {
			newGroup := &TargetGroup{
				Targets: []model.LabelSet{{model.AddressLabel: target[model.AddressLabel]}},
				Labels:  group.Labels.Merge(target),
			}
			// We did a full merge, so we need to delete the one label we don't want
			// to copy over.
			delete(newGroup.Labels, model.AddressLabel)

			apiEnts = append(apiEnts, newGroup)
		}
	}

	rw.Header().Add("Content-Type", "application/json")
	_ = json.NewEncoder(rw).Encode(apiEnts)
}
