// Package cluster implements a clustering mechanism for Grafana Agents.
package cluster

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rfratto/croissant/id"
	"github.com/rfratto/croissant/node"
	"google.golang.org/grpc"
)

// Config controls the clustering of Agents.
type Config struct {
	// NodeName is the name used to identify the loacl agent in the cluster.
	NodeName string

	AdvertiseAddr       string
	AdvertiseInterfaces flagext.StringSlice
	AdvertisePort       int

	// List of Peers to join.
	Peers flagext.StringSlice
}

// DefaultConfig holds default options for clustering.
var DefaultConfig = Config{
	AdvertiseInterfaces: []string{"eth0", "en0"},
}

func (c *Config) RegisterFlags(f *flag.FlagSet) {
	*c = DefaultConfig

	f.StringVar(&c.NodeName, "cluster.node-name", DefaultConfig.NodeName, "Name to identify node in cluster. If empty, defaults to hostname.")

	f.StringVar(&c.AdvertiseAddr, "cluster.advertise-addr", DefaultConfig.AdvertiseAddr, "IP address to advertise to peers. If not set, defaults to the first IP found from cluster.advertise-interfaces.")
	f.Var(&c.AdvertiseInterfaces, "cluster.advertise-interfaces", "Interfaces to use for discovering advertise address. Mutually exclusive with setting cluster.advertise-addr.")

	f.Var(&c.Peers, "cluster.join-peers", "List of peers to join when starting up.")
}

// Cluster is a cluster of Agents using a hash ring to distribute work.
type Cluster struct {
	cfg  Config
	log  log.Logger
	node *node.Node
	id   id.ID

	started      chan struct{}
	closeStarted sync.Once
}

// New creates a new Cluster.
func New(l log.Logger, reg prometheus.Registerer, cfg Config) (*Cluster, error) {
	if l == nil {
		l = log.NewNopLogger()
	}
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	l = log.With(l, "component", "cluster")

	if cfg.NodeName == "" {
		hn, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to generate node name: %w", err)
		}
		cfg.NodeName = hn
	}

	if cfg.AdvertiseAddr == "" {
		var err error
		cfg.AdvertiseAddr, err = util.GetFirstAddressOf(cfg.AdvertiseInterfaces)
		if err != nil {
			return nil, fmt.Errorf("failed to get advertise address: %w", err)
		}
	}

	nID := id.NewGenerator(32).Get(cfg.NodeName)

	n, err := node.New(
		node.Config{
			ID:            nID,
			BroadcastAddr: fmt.Sprintf("%s:%d", cfg.AdvertiseAddr, cfg.AdvertisePort),
			Log:           l,
		},
		noopApplication{},
		grpc.WithInsecure(), // TODO(rfratto): this won't work if agent is running with TLS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return &Cluster{
		id:   nID,
		cfg:  cfg,
		log:  l,
		node: n,

		started: make(chan struct{}),
	}, nil
}

// WaitJoined returns when the node has joined the cluster.
func (c *Cluster) WaitJoined() {
	<-c.started
}

// Join starts the Cluster.
func (c *Cluster) Join(ctx context.Context) error {
	defer c.closeStarted.Do(func() { close(c.started) })

	level.Info(c.log).Log("msg", "joining cluster", "peers", strings.Join(c.cfg.Peers, ","))
	return c.node.Join(ctx, c.cfg.Peers)
}

// Node returns the local node for the cluster.
func (c *Cluster) Node() *node.Node { return c.node }

// ID returns the ID of this node.
func (c *Cluster) ID() id.ID { return c.id }

// WireGRPC hooks up the cluster to the gRPC server allowing nodes to contact
// each other.
func (c *Cluster) WireGRPC(s *grpc.Server) {
	c.node.Register(s)
}

// ServeHTTP serves the state of the cluster as an HTTP page.
func (c *Cluster) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	node.WriteHTTPState(c.log, rw, c.node)
}

// Close closes the cluster.
func (c *Cluster) Close() error {
	level.Info(c.log).Log("msg", "closing cluster connection")
	return c.node.Close()
}

type noopApplication struct{}

func (noopApplication) PeersChanged([]node.Peer) {}
