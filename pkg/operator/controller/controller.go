package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/agent/pkg/operator"
	monitoringv1 "github.com/grafana/agent/pkg/operator/apis/monitoring/v1"
	monitoringclient "github.com/grafana/agent/pkg/operator/client/versioned"
	agentinformers "github.com/grafana/agent/pkg/operator/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/k8sutil"
	promoperator "github.com/prometheus-operator/prometheus-operator/pkg/operator"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// Controller manages the lifecycle of Grafana Agent deployments and
// monitoring configurations.
type Controller struct {
	log     log.Logger
	kclient kubernetes.Interface
	mclient monitoringclient.Interface

	agentInfs *informers.ForResource

	queue workqueue.RateLimitingInterface
}

// New creates a new Controller.
func New(ctx context.Context, cfg operator.Config, log log.Logger, r prometheus.Registerer) (*Controller, error) {
	var (
		clusterCfg *rest.Config
		err        error
	)

	if cfg.APIServer.UseKubeConfig {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		clusterCfg, err = kubeConfig.ClientConfig()
	} else {
		clusterCfg, err = k8sutil.NewClusterConfig(cfg.APIServer.Addr, cfg.APIServer.SkipVerify, &rest.TLSClientConfig{
			CertFile: cfg.APIServer.CertFile,
			KeyFile:  cfg.APIServer.KeyFile,
			CAFile:   cfg.APIServer.CAFile,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cluster config: %w", err)
	}

	kclient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}

	// TODO(rfratto): this client won't be enough for Prometheus operator CRDs, we'll need a tertiary client here.
	mclient, err := monitoringclient.NewForConfig(clusterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring client: %w", err)
	}

	c := &Controller{
		log:     log,
		kclient: kclient,
		mclient: mclient,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "grafana-agent"),
	}

	c.agentInfs, err = informers.NewInformersForResource(
		agentinformers.NewMonitoringInformerFactories(
			nil, // TODO(rfratto): fill out
			nil, // TODO(rfratto): fill out
			mclient,
			5*time.Minute,
			func(options *v1.ListOptions) {
				options.LabelSelector = ""
			},
		),
		monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.GrafanaAgentName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent informers: %w", err)
	}

	// TODO(rfratto): other informers

	return c, nil
}

// Run runs the controller until ctx is canceled or an error occurs.
func (c *Controller) Run(ctx context.Context) error {
	errCh := make(chan error)
	go func() {
		v, err := c.kclient.Discovery().ServerVersion()
		if err != nil {
			errCh <- fmt.Errorf("communicating with server failed: %w", err)
			return
		}
		level.Info(c.log).Log("msg", "connection established", "cluster-version", v)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
		level.Info(c.log).Log("msg", "CRD API endpoints ready")
	case <-ctx.Done():
		return nil
	}

	// TODO(rfratto): start workers, do other setup
	go c.worker(ctx)

	go c.agentInfs.Start(ctx.Done())

	if err := c.waitForCacheSync(ctx); err != nil {
		return err
	}
	c.addHandlers()

	// TODO(rfratto): kubelet sync enabled?

	<-ctx.Done()
	return nil
}

// worker runs a worker thread that just dequeues items, processes them, and
// marks them as done. It enforces that the syncHandler is never invoked
// concurrently with the same key.
func (c *Controller) worker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem waits for the next item in the queue. Returns false when
// the worker should exit.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	// TODO(rfratto): reconcile counter
	err := c.sync(ctx, key.(string))
	// TODO(rfratto): set sync status for key
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	// TODO(rfratto): reconcile errors counter
	// TODO(rfratto): handle error? utilruntime.HandleError
	c.queue.AddRateLimited(key)

	return true
}

func (c *Controller) sync(ctx context.Context, key string) error {
	agentObj, err := c.agentInfs.Get(key)

	if apierrors.IsNotFound(err) {
		// TODO(rfratto): metrics forget key
		return nil
	}
	if err != nil {
		return err
	}

	a := agentObj.(*monitoringv1.GrafanaAgent)
	a = a.DeepCopy()
	a.APIVersion = monitoringv1.SchemeGroupVersion.String()
	a.Kind = monitoringv1.GrafanaAgentsKind

	level.Info(c.log).Log("msg", "sync grafana agent", "key", key)
	// TODO(rfratto): this is where the meat of the operator happens. Do everything here.

	_ = ctx
	return nil
}

func (c *Controller) waitForCacheSync(ctx context.Context) error {
	var failed bool

	for _, infs := range []struct {
		name                 string
		informersForResource *informers.ForResource
	}{
		{"GrafanaAgent", c.agentInfs},
	} {
		for _, inf := range infs.informersForResource.GetInformers() {
			if !promoperator.WaitForNamedCacheSync(ctx, "grafana-agent", log.With(c.log, "informer", infs.name), inf.Informer()) {
				failed = true
			}
		}
	}

	// TODO(rfratto): SharedIndexInformer

	if failed {
		return fmt.Errorf("failed to sync caches")
	}

	level.Info(c.log).Log("msg", "successfully synced all caches")
	return nil
}

func (c *Controller) addHandlers() {
	// NOTE(rfratto): while the handlers here directly enqueue items, nearly no other handler
	// works this way. Instead, other handlers try to find the related top-level CRDs and re-enqueue
	// the key for that Agent CRD, causing it to be re-applied.
	c.agentInfs.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAgentAdd,
		DeleteFunc: c.handleAgentDelete,
		UpdateFunc: c.handleAgentUpdate,
	})
}

func (c *Controller) keyFunc(obj interface{}) (string, bool) {
	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		level.Error(c.log).Log("msg", "creating key failed", "err", err)
		return k, false
	}
	return k, true
}

func (c *Controller) handleAgentAdd(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	level.Debug(c.log).Log("msg", "GrafanaAgent added", "key", key)
	c.enqueue(key)
}

func (c *Controller) handleAgentDelete(obj interface{}) {
	key, ok := c.keyFunc(obj)
	if !ok {
		return
	}

	level.Debug(c.log).Log("msg", "GrafanaAgent deleted", "key", key)
	c.enqueue(key)
}

func (c *Controller) handleAgentUpdate(old, cur interface{}) {
	if old.(*monitoringv1.GrafanaAgent).ResourceVersion == cur.(*monitoringv1.GrafanaAgent).ResourceVersion {
		return
	}

	key, ok := c.keyFunc(cur)
	if !ok {
		return
	}

	level.Debug(c.log).Log("msg", "GrafanaAgent updated", "key", key)
	c.enqueue(key)
}

// enqueue adds a key to the key. If obj is a key already it gets added
// directly. Otherwise, the key is extracted via keyFunc.
func (c *Controller) enqueue(obj interface{}) {
	if obj == nil {
		return
	}

	key, ok := obj.(string)
	if !ok {
		key, ok = c.keyFunc(obj)
		if !ok {
			return
		}
	}

	c.queue.Add(key)
}
