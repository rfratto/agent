package main

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/agent/cmd/agent-operator/internal/logutil"
	"github.com/grafana/agent/pkg/operator"
	grafana_v1alpha1 "github.com/grafana/agent/pkg/operator/apis/monitoring/v1alpha1"
	"github.com/grafana/agent/pkg/operator/assets"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"gopkg.in/yaml.v2"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controller "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client.Client
	scheme *runtime.Scheme

	// Various event handlers that can trigger reconciliation for watched
	// resources.
	eventPromInstances  *operator.EnqueueRequestForSelector
	eventServiceMonitor *operator.EnqueueRequestForSelector
	eventPodMonitor     *operator.EnqueueRequestForSelector
	eventProbe          *operator.EnqueueRequestForSelector
	eventSecret         *operator.EnqueueRequestForSelector
}

func (r *reconciler) Reconcile(ctx context.Context, req controller.Request) (controller.Result, error) {
	l := logutil.FromContext(ctx)
	level.Info(l).Log("msg", "reconciling grafana-agent")

	var agent grafana_v1alpha1.GrafanaAgent
	if err := r.Get(ctx, req.NamespacedName, &agent); errors.IsNotFound(err) {
		level.Debug(l).Log("msg", "detected deleted agent, cleaning up watchers")

		r.eventPromInstances.Notify(req.NamespacedName, nil)
		r.eventServiceMonitor.Notify(req.NamespacedName, nil)
		r.eventPodMonitor.Notify(req.NamespacedName, nil)
		r.eventProbe.Notify(req.NamespacedName, nil)
		r.eventSecret.Notify(req.NamespacedName, nil)

		return controller.Result{}, nil
	} else if err != nil {
		level.Error(l).Log("msg", "unable to get grafana-agent", "err", err)
		return controller.Result{}, nil
	}

	secrets := make(assets.SecretStore)
	builder := deploymentBuilder{
		Client:            r.Client,
		Agent:             &agent,
		Secrets:           secrets,
		ResourceSelectors: make(map[string][]operator.ResourceSelector),
	}

	deployment, err := builder.Build(l, ctx)
	if err != nil {
		level.Error(l).Log("msg", "unable to collect resources", "err", err)
		return controller.Result{}, nil
	}

	// Create configuration in a secret
	{
		rawConfig, err := deployment.BuildConfig(secrets)
		if err != nil {
			level.Error(l).Log("msg", "unable to build config", "err", err)
			return controller.Result{}, nil
		}

		configBytes, err := yaml.Marshal(rawConfig)
		if err != nil {
			level.Error(l).Log("msg", "unable to marshal config", "err", err)
			return controller.Result{}, nil
		}

		blockOwnerDeletion := true

		secret := core_v1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Namespace: agent.Namespace,
				Name:      fmt.Sprintf("%s-config", agent.Name),
				OwnerReferences: []v1.OwnerReference{{
					APIVersion:         agent.APIVersion,
					BlockOwnerDeletion: &blockOwnerDeletion,
					Kind:               agent.Kind,
					Name:               agent.Name,
					UID:                agent.UID,
				}},
			},
			Data: map[string][]byte{"agent.yml": configBytes},
		}

		level.Info(l).Log("msg", "creating or updating secret", "secret", secret.Name)
		err = r.Client.Update(ctx, &secret)
		if errors.IsNotFound(err) {
			err = r.Client.Create(ctx, &secret)
		}
		if errors.IsAlreadyExists(err) {
			return controller.Result{Requeue: true}, nil
		}
		if err != nil {
			level.Error(l).Log("msg", "failed to create Secret for storing config", "err", err)
			return controller.Result{Requeue: true}, nil
		}
	}

	// Create secrets from asset store. These will be used to create volume
	// mounts into pods.
	{
		blockOwnerDeletion := true

		data := make(map[string][]byte)
		for k, value := range secrets {
			data[operator.SanitizeLabelName(string(k))] = []byte(value)
		}

		secret := core_v1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Namespace: agent.Namespace,
				Name:      fmt.Sprintf("%s-secrets", agent.Name),
				OwnerReferences: []v1.OwnerReference{{
					APIVersion:         agent.APIVersion,
					BlockOwnerDeletion: &blockOwnerDeletion,
					Kind:               agent.Kind,
					Name:               agent.Name,
					UID:                agent.UID,
				}},
			},
			Data: data,
		}

		level.Info(l).Log("msg", "creating or updating secret", "secret", secret.Name)
		err = r.Client.Update(ctx, &secret)
		if errors.IsNotFound(err) {
			err = r.Client.Create(ctx, &secret)
		}
		if errors.IsAlreadyExists(err) {
			return controller.Result{Requeue: true}, nil
		}
		if err != nil {
			level.Error(l).Log("msg", "failed to create Secret for storing config", "err", err)
			return controller.Result{Requeue: true}, nil
		}
	}

	// Generate and create StatefulSet.
	{
		ss, err := generateStatefulSet(deployment, secrets)
		if err != nil {
			level.Error(l).Log("msg", "failed to generate statefulset", "err", err)
			return controller.Result{Requeue: false}, nil
		}

		level.Info(l).Log("msg", "creating or updating statefulset", "statefulset", ss.Name)
		err = r.Client.Update(ctx, ss)
		if errors.IsNotFound(err) {
			err = r.Client.Create(ctx, ss)
		}
		if errors.IsAlreadyExists(err) {
			return controller.Result{Requeue: true}, nil
		}
		if err != nil {
			level.Error(l).Log("msg", "failed to create statefulset", "err", err)
			return controller.Result{}, nil
		}
	}

	// Update our notifiers with every object we discovered. This ensures that we
	// will re-reconcile whenever any of these objects (which composed our configs)
	// changes.
	r.eventPromInstances.Notify(req.NamespacedName, builder.ResourceSelectors[promResourceKey])
	r.eventServiceMonitor.Notify(req.NamespacedName, builder.ResourceSelectors[smResourceKey])
	r.eventPodMonitor.Notify(req.NamespacedName, builder.ResourceSelectors[pmResourceKey])
	r.eventProbe.Notify(req.NamespacedName, builder.ResourceSelectors[probeResourceKey])
	r.eventSecret.Notify(req.NamespacedName, builder.ResourceSelectors[secretResoruceKey])

	return controller.Result{}, nil
}

const (
	promResourceKey   = "prom"
	smResourceKey     = "sm"
	pmResourceKey     = "pm"
	probeResourceKey  = "probe"
	secretResoruceKey = "secret"
)

type deploymentBuilder struct {
	client.Client

	Agent   *grafana_v1alpha1.GrafanaAgent
	Secrets assets.SecretStore

	// ResourceSelectors is filled as objects are found and can be used to
	// trigger future reconciles.
	ResourceSelectors map[string][]operator.ResourceSelector
}

func (b *deploymentBuilder) Build(l log.Logger, ctx context.Context) (operator.Deployment, error) {
	instances, err := b.getPrometheusInstances(ctx)
	if err != nil {
		return operator.Deployment{}, err
	}
	promInstances := make([]operator.PrometheusInstance, 0, len(instances))

	for _, inst := range instances {
		sMons, err := b.getServiceMonitors(ctx, inst)
		if err != nil {
			return operator.Deployment{}, fmt.Errorf("unable to fetch ServiceMonitors: %w", err)
		}
		pMons, err := b.getPodMonitors(ctx, inst)
		if err != nil {
			return operator.Deployment{}, fmt.Errorf("unable to fetch PodMonitors: %w", err)
		}
		probes, err := b.getProbes(ctx, inst)
		if err != nil {
			return operator.Deployment{}, fmt.Errorf("unable to fetch Probes: %w", err)
		}

		// TODO(rfratto): load in used secrets from instance into secrets store
		// TODO(rfratto): load in used secrets from sMons, pMons, probes

		promInstances = append(promInstances, operator.PrometheusInstance{
			Instance:        inst,
			ServiceMonitors: sMons,
			PodMonitors:     pMons,
			Probes:          probes,
		})
	}

	return operator.Deployment{
		Agent:      b.Agent,
		Prometheis: promInstances,
	}, nil
}

func (b *deploymentBuilder) getPrometheusInstances(ctx context.Context) ([]*grafana_v1alpha1.PrometheusInstance, error) {
	sel, err := b.getResourceSelector(
		b.Agent.Namespace,
		b.Agent.Spec.Prometheus.InstanceNamespaceSelector,
		b.Agent.Spec.Prometheus.InstanceSelector,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to build prometheus resource selector: %w", err)
	}
	b.ResourceSelectors[promResourceKey] = append(b.ResourceSelectors[promResourceKey], sel)

	var (
		list        grafana_v1alpha1.PrometheusInstanceList
		namespace   = namespaceFromSelector(sel)
		listOptions = &client.ListOptions{LabelSelector: sel.Labels, Namespace: namespace}
	)
	if err := b.List(ctx, &list, listOptions); err != nil {
		return nil, err
	}

	items := make([]*grafana_v1alpha1.PrometheusInstance, 0, len(list.Items))
	for _, item := range list.Items {
		if match, err := b.matchNamespace(ctx, &item.ObjectMeta, sel); match {
			items = append(items, item)
		} else if err != nil {
			return nil, fmt.Errorf("failed getting namespace: %w", err)
		}
	}
	return items, nil
}

func (b *deploymentBuilder) getResourceSelector(
	currentNamespace string,
	namespaceSelector *v1.LabelSelector,
	objectSelector *v1.LabelSelector,
) (sel operator.ResourceSelector, err error) {

	// Set up our namespace label and object label selectors. By default, we'll
	// match everything (the inverse of the k8s default). If we specify anything,
	// we'll narrow it down.
	var (
		nsLabels  = labels.Everything()
		objLabels = labels.Everything()
	)
	if namespaceSelector != nil {
		nsLabels, err = v1.LabelSelectorAsSelector(namespaceSelector)
		if err != nil {
			return sel, err
		}
	}
	if objectSelector != nil {
		objLabels, err = v1.LabelSelectorAsSelector(objectSelector)
		if err != nil {
			return sel, err
		}
	}

	sel = operator.ResourceSelector{
		NamespaceName: prom.NamespaceSelector{
			MatchNames: []string{currentNamespace},
		},
		NamespaceLabels: nsLabels,
		Labels:          objLabels,
	}

	// If we have a namespace selector, that means we're matching more than one
	// namespace and we should adjust NamespaceName appropriatel.
	if namespaceSelector != nil {
		sel.NamespaceName = prom.NamespaceSelector{Any: true}
	}

	return
}

// namespaceFromSelector returns the namespace string that should be used for
// querying lists of objects. If the ResourceSelector is looking at more than
// one namespace, an empty string will be returned. Otherwise, it will return
// the first namespace.
func namespaceFromSelector(sel operator.ResourceSelector) string {
	if !sel.NamespaceName.Any && len(sel.NamespaceName.MatchNames) == 1 {
		return sel.NamespaceName.MatchNames[0]
	}
	return ""
}

func (b *deploymentBuilder) matchNamespace(
	ctx context.Context,
	obj *v1.ObjectMeta,
	sel operator.ResourceSelector,
) (bool, error) {
	// If we were matching on a specific namespace, there's no
	// further work to do here.
	if namespaceFromSelector(sel) != "" {
		return true, nil
	}

	var ns core_v1.Namespace
	if err := b.Get(ctx, types.NamespacedName{Name: obj.Namespace}, &ns); err != nil {
		return false, fmt.Errorf("failed getting namespace: %w", err)
	}

	return sel.NamespaceLabels.Matches(labels.Set(ns.Labels)), nil
}

func (b *deploymentBuilder) getServiceMonitors(
	ctx context.Context,
	inst *grafana_v1alpha1.PrometheusInstance,
) ([]*prom.ServiceMonitor, error) {
	sel, err := b.getResourceSelector(
		inst.Namespace,
		inst.Spec.ServiceMonitorNamespaceSelector,
		inst.Spec.ServiceMonitorSelector,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to build service monitor resource selector: %w", err)
	}
	b.ResourceSelectors[smResourceKey] = append(b.ResourceSelectors[smResourceKey], sel)

	var (
		list        prom.ServiceMonitorList
		namespace   = namespaceFromSelector(sel)
		listOptions = &client.ListOptions{LabelSelector: sel.Labels, Namespace: namespace}
	)
	if err := b.List(ctx, &list, listOptions); err != nil {
		return nil, err
	}

	items := make([]*prom.ServiceMonitor, 0, len(list.Items))
	for _, item := range list.Items {
		if match, err := b.matchNamespace(ctx, &item.ObjectMeta, sel); match {
			items = append(items, item)
		} else if err != nil {
			return nil, fmt.Errorf("failed getting namespace: %w", err)
		}
	}
	return items, nil
}

func (b *deploymentBuilder) getPodMonitors(
	ctx context.Context,
	inst *grafana_v1alpha1.PrometheusInstance,
) ([]*prom.PodMonitor, error) {
	sel, err := b.getResourceSelector(
		inst.Namespace,
		inst.Spec.PodMonitorNamespaceSelector,
		inst.Spec.PodMonitorSelector,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to build service monitor resource selector: %w", err)
	}
	b.ResourceSelectors[pmResourceKey] = append(b.ResourceSelectors[pmResourceKey], sel)

	var (
		list        prom.PodMonitorList
		namespace   = namespaceFromSelector(sel)
		listOptions = &client.ListOptions{LabelSelector: sel.Labels, Namespace: namespace}
	)
	if err := b.List(ctx, &list, listOptions); err != nil {
		return nil, err
	}

	items := make([]*prom.PodMonitor, 0, len(list.Items))
	for _, item := range list.Items {
		if match, err := b.matchNamespace(ctx, &item.ObjectMeta, sel); match {
			items = append(items, item)
		} else if err != nil {
			return nil, fmt.Errorf("failed getting namespace: %w", err)
		}
	}
	return items, nil
}

func (b *deploymentBuilder) getProbes(
	ctx context.Context,
	inst *grafana_v1alpha1.PrometheusInstance,
) ([]*prom.Probe, error) {
	sel, err := b.getResourceSelector(
		inst.Namespace,
		inst.Spec.ProbeNamespaceSelector,
		inst.Spec.ProbeSelector,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to build service monitor resource selector: %w", err)
	}
	b.ResourceSelectors[probeResourceKey] = append(b.ResourceSelectors[probeResourceKey], sel)

	var namespace string
	if !sel.NamespaceName.Any && len(sel.NamespaceName.MatchNames) == 1 {
		namespace = sel.NamespaceName.MatchNames[0]
	}

	var list prom.ProbeList
	listOptions := &client.ListOptions{LabelSelector: sel.Labels, Namespace: namespace}
	if err := b.List(ctx, &list, listOptions); err != nil {
		return nil, err
	}

	items := make([]*prom.Probe, 0, len(list.Items))
	for _, item := range list.Items {
		if match, err := b.matchNamespace(ctx, &item.ObjectMeta, sel); match {
			items = append(items, item)
		} else if err != nil {
			return nil, fmt.Errorf("failed getting namespace: %w", err)
		}
	}
	return items, nil
}
