package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/agent/pkg/cluster"
	"github.com/grafana/agent/pkg/discovery/discoverypb"
	"github.com/rfratto/ckit"
	"github.com/rfratto/ckit/peer"
	"github.com/rfratto/ckit/shard"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// TODO(rfratto): have n implement client pool stuff so we don't have to pass
// around both all the time

type NodeConfig struct {
	// Names of discovery jobs that are sharded amongst the cluster (i.e., only
	// one node is performing that discovery).
	//
	// Nodes will only broadcast and aggregate targets for jobs that appear in
	// this list; all other targets will be ignored.
	ShardedJobs []string
}

// Node enables sharing discovered targets amongst a cluster of peers. In a
// cluster, some discovery jobs may be sharded amongst the cluster, where only
// one Node is responsible for running the actual job.
//
// Nodes connect to each other to share discovered targets, synchronizing with
// a Registry to build a definitive set of cluster-wide targets.
type Node struct {
	log log.Logger
	r   *Registry
	c   cluster.Node

	ctx       context.Context
	cancel    context.CancelFunc
	reshardCh chan struct{}
	exitedCh  chan struct{}
	conns     map[peer.Peer]context.CancelFunc

	mut         sync.RWMutex
	cfg         NodeConfig
	shardedJobs map[string]struct{}
}

// NewNode creates a new Node which synchronizes discoverd targets to r.
// Connections to peers are made through c.
func NewNode(l log.Logger, r *Registry, c cluster.Node, cfg NodeConfig) (n *Node, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	n = &Node{
		log: l,
		r:   r,
		c:   c,

		ctx:       ctx,
		cancel:    cancel,
		reshardCh: make(chan struct{}, 1),
		conns:     make(map[peer.Peer]context.CancelFunc),
	}

	// Register n to watch for changes to the cluster state.
	c.Observe(ckit.FuncObserver(func(_ []peer.Peer) (reregister bool) {
		select {
		case <-n.exitedCh:
			return false
		default:
			n.queueReshard()
			return true
		}
	}))

	return n, n.UpdateConfig(cfg)
}

func (n *Node) queueReshard() {
	select {
	case n.reshardCh <- struct{}{}:
	default:
	}
}

func (n *Node) run() {
	defer close(n.exitedCh)

	// Cancel all connections to peers on shutdown.
	defer func() {
		for p, cancel := range n.conns {
			cancel()
			delete(n.conns, p)
		}
	}()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-n.reshardCh:
			n.reshard()
		}
	}
}

func (n *Node) reshard() {
	n.mut.RLock()
	jobs := n.cfg.ShardedJobs
	n.mut.RUnlock()

	expectedPeers := map[peer.Peer]struct{}{}

	for _, job := range jobs {
		ps, err := n.c.Lookup(shard.StringKey(job), 1, shard.OpReadWrite)
		if err != nil {
			level.Error(n.log).Log("msg", "failed to find owner for sharded SD job", "job", job, "err", err)
			continue
		}
		if ps[0].Self {
			// Ignore ourselves because our local node is already publishing targets to the registry.
			continue
		}
		expectedPeers[ps[0]] = struct{}{}
	}

	// Shut down connections that shouldn't be open anymore.
	for p, cancel := range n.conns {
		if _, expected := expectedPeers[p]; expected {
			continue
		}
		cancel()
		delete(n.conns, p)
	}

	// Start new connections as needed.
	for p := range expectedPeers {
		if _, exists := n.conns[p]; exists {
			continue
		}

		ctx, cancel := context.WithCancel(n.ctx)
		n.conns[p] = cancel
		go n.watchPeer(ctx, p)
	}
}

func (n *Node) watchPeer(ctx context.Context, p peer.Peer) {
	defer level.Debug(n.log).Log("msg", "closed SD target stream connection to peer", "peer", p)

	wait := func() {
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
		}
	}

	var buf bytes.Buffer

	// Our connection to p may die while streaming; we want to continually
	// reconnect unless our parent context has been canceled.
ConnLoop:
	for {
		if ctx.Err() != nil {
			return
		}

		level.Debug(n.log).Log("msg", "opening SD target stream connection to peer", "peer", p)

		// TODO(rfratto): get via pool
		var cc *grpc.ClientConn
		cli := discoverypb.NewDiscovererClient(cc)

		stream, err := cli.Watch(ctx, &discoverypb.WatchRequest{})
		if err != nil {
			level.Warn(n.log).Log("msg", "failed to establish SD target stream connection to peer", "peer", p, "err", err)
			wait()
			continue ConnLoop
		}

		for {
			msg, err := stream.Recv()
			if ctx.Err() != nil {
				return
			} else if err != nil {
				level.Warn(n.log).Log("msg", "SD target stream connection to peer failed", "peer", p, "err", err)
				wait()
				continue ConnLoop
			}

			ts := TargetSet{ID: msg.GetId()}

			buf.Reset()
			err = json.NewEncoder(&buf).Encode(msg.GetGroups())
			if err != nil {
				level.Error(n.log).Log("msg", "failed to transcode client TargetSet", "step", "decode_from_proto", "err", err)
				continue
			}

			err = json.NewDecoder(&buf).Decode(&ts.Groups)
			if err != nil {
				level.Error(n.log).Log("msg", "failed to transcode client TargetSet", "step", "encode_to_groups", "err", err)
				continue
			}

			// It's possible that ts may be for a job that p isn't responsible for
			// anymore. However, this should be OK: the cluster state will eventually
			// settle down and any contention between TargetSets will stop.
			//
			// Allowing TargetSet messages from improper owners while a cluster state
			// change is propagating also ensures that we continue to receive updates
			// from *someone*, allowing us to pick up new targets faster.
			n.r.Update(ts)
		}
	}
}

// UpdateConfig updates the configuration of the Node.
func (n *Node) UpdateConfig(c NodeConfig) error {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.shardedJobs = make(map[string]struct{}, len(c.ShardedJobs))
	for _, j := range c.ShardedJobs {
		n.shardedJobs[j] = struct{}{}
	}

	n.cfg = c
	n.queueReshard()
	return nil
}

// RegisterService registers n as a gRPC service for sharing targets from
// sharded SD jobs.
func (n *Node) RegisterService(srv *grpc.Server) {
	discoverypb.RegisterDiscovererServer(srv, &nodeDelegate{Node: n})
}

// Close closes n.
func (n *Node) Close() error {
	n.cancel()
	<-n.exitedCh
	return nil
}

type nodeDelegate struct {
	discoverypb.UnimplementedDiscovererServer
	*Node
}

// Watch implements discoverypb.DiscovererServer.
func (n *nodeDelegate) Watch(_ *discoverypb.WatchRequest, stream discoverypb.Discoverer_WatchServer) error {
	// TODO(rfratto): if UpdateConfig is called we need to cancel the
	// subscription to the registry and re-start it. It's possible that the list
	// of sharded jobs changed and may never emit a new TargetSet. Restarting the
	// subscription avoids dropping the TargetSet at the cost of re-broadcasting
	// other TargetSets.

	// TODO(rfratto): we should also restart Watch every 60 seconds to ensure
	// that targets eventually propagate. This will help prevent things which
	// never update their targets (like static configs) to avoid being rendered
	// permanently incorrect by ordering issues while a cluster state change is
	// propagating.

	var buf bytes.Buffer

	return n.r.Watch(stream.Context(), func(_ context.Context, ts TargetSet) error {
		n.mut.RLock()
		defer n.mut.RUnlock()

		// We're only trying to emit targets for sharded jobs that we own, so
		// ignore everything else.
		if _, sharded := n.shardedJobs[ts.ID]; !sharded || !n.ownsKey(ts.ID) {
			return nil
		}

		buf.Reset()

		err := json.NewEncoder(&buf).Encode(ts.Groups)
		if err != nil {
			level.Error(n.log).Log("msg", "failed to transcode TargetSet", "step", "encode_to_json", "err", err)
			return nil
		}

		var protoStruct structpb.Struct
		if err := json.NewDecoder(&buf).Decode(&protoStruct); err != nil {
			level.Error(n.log).Log("msg", "failed to transcode TargetSet", "step", "decode_to_proto", "err", err)
			return nil
		}

		return stream.Send(&discoverypb.TargetSet{
			Id:     ts.ID,
			Groups: &protoStruct,
		})
	})
}

func (n *nodeDelegate) ownsKey(id string) bool {
	ps, err := n.c.Lookup(shard.StringKey(id), 1, shard.OpRead)
	if err != nil {
		return true
	}
	return ps[0].Self
}
