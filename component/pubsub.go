package component

import "sync"

type Pubsub[Message comparable] struct {
	mut      sync.RWMutex
	watchers map[Message]struct{}
}

func NewPubsub[Message comparable]() *Pubsub[Message] {
	return &Pubsub[Message]{
		watchers: make(map[Message]struct{}),
	}
}

func (ps *Pubsub[Message]) List() []Message {
	ps.mut.RLock()
	defer ps.mut.RUnlock()

	res := make([]Message, 0, len(ps.watchers))
	for m := range ps.watchers {
		res = append(res, m)
	}
	return res
}

func (ps *Pubsub[Message]) Subscribe(m Message) {
	ps.mut.Lock()
	defer ps.mut.Unlock()
	ps.watchers[m] = struct{}{}
}

func (ps *Pubsub[Message]) Unsubscribe(m Message) {
	ps.mut.Lock()
	defer ps.mut.Unlock()
	delete(ps.watchers, m)
}
