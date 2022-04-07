package discovery

import (
	"context"
	"sync"

	"github.com/prometheus/prometheus/discovery/targetgroup"
)

// Registry is a global Registry of TargetSets. A Registry can be watched for
// changes, notifiying subscribers whenever a TargetSet is updated.
type Registry struct {
	mut     sync.RWMutex
	entries map[string]entry
	nextID  uint64

	watchers sync.Map
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]entry),
	}
}

type entry struct {
	UpdatedAt uint64
	Set       TargetSet
}

// TargetSet is a set of targets associated with a specific ID.
type TargetSet struct {
	ID     string
	Groups []*targetgroup.Group
}

func (c *Registry) Watch(ctx context.Context, f func(context.Context, TargetSet) error) error {
	// Create the watcherCh and immediately enqueue an event so we calculate
	// current state.
	watcherCh := make(chan struct{}, 1)
	watcherCh <- struct{}{}

	c.watchers.Store(watcherCh, struct{}{})
	defer c.watchers.Delete(watcherCh)

	var lastID uint64

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watcherCh:

			var (
				updated  []TargetSet
				newestID = lastID
			)
			for _, ent := range c.currentEntries() {
				if ent.UpdatedAt <= lastID {
					continue
				}
				if ent.UpdatedAt > newestID {
					newestID = ent.UpdatedAt
				}

				updated = append(updated, ent.Set)
			}
			lastID = newestID

			for _, ts := range updated {
				err := f(ctx, ts)
				if err != nil && ctx.Err() == nil {
					// Only return the error when our context is still valid.
					return err
				}
			}
		}
	}
}

// Update adds ts into the cache. Watchers will be notified.
func (c *Registry) Update(ts TargetSet) {
	c.mut.Lock()
	c.nextID++
	c.entries[ts.ID] = entry{
		UpdatedAt: c.nextID,
		Set:       ts,
	}
	c.mut.Unlock()

	// Queue updates to all of our watchers.
	c.watchers.Range(func(key, _ interface{}) bool {
		select {
		case key.(chan struct{}) <- struct{}{}:
			// The watcher was waiting for a new event; we just queued a refresh.
		default:
			// The watcher already has a pending update; do nothing.
		}
		return true
	})
}

// Delete removes a Registry entry by its ID. Watchers do not get notified
// about deleted entries.
func (c *Registry) Delete(id string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.entries, id)
}

// TargetSets returns the current TargetSets in the cache. Order of returned
// TargetSets is not guaranteed.
func (c *Registry) TargetSets() []TargetSet {
	panic("NYI")
}

// currentEntries returns a shallow clone of the current entries.
func (c *Registry) currentEntries() []entry {
	c.mut.RLock()
	defer c.mut.RUnlock()

	ents := make([]entry, 0, len(c.entries))
	for _, ent := range c.entries {
		ents = append(ents, ent)
	}
	return ents
}
