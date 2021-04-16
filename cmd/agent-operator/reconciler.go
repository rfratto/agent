package main

import (
	"context"
	"fmt"

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

	var (
		promResources   []operator.ResourceSelector
		smResources     []operator.ResourceSelector
		pmResources     []operator.ResourceSelector
		probeResources  []operator.ResourceSelector
		secretResources []operator.ResourceSelector
	)

	// First, find all the Prometheus instances custom resources.
	instanceResources, instanceSelector, err := r.prometheusInstances(
		ctx,
		agent.Namespace,
		agent.Spec.Prometheus.InstanceNamespaceSelector,
		agent.Spec.Prometheus.InstanceSelector,
	)
	if err != nil {
		level.Error(l).Log("msg", "unable to get prometheus instances", "err", err)
		return controller.Result{}, nil
	}
	promResources = append(promResources, instanceSelector)

	// Create a secrets store for all consumed secrets. The secrets are used for
	// in-memory references when building the config, but will also be converted
	// into a giant Secret to be used by the GrafanaAgent pods.
	secrets := make(assets.SecretStore)

	// Now, for each instance resource we discovered, we have to find all
	// associated:
	//
	// 1. ServiceMonitors
	// 2. PodMonitors
	// 3. Probes
	//
	// We also have to populate the secrets store for all discovered assets,
	// including for the instance CR itself.
	var prometheusInstances []operator.PrometheusInstance

	for _, inst := range instanceResources {
		level.Info(l).Log("msg", "discovered instance", "instance", inst.Namespace+"/"+inst.Name)

		// TODO(rfratto): load in used secrets from instance

		sMons, smSelector, err := r.serviceMonitors(
			ctx,
			inst.Namespace,
			inst.Spec.ServiceMonitorNamespaceSelector,
			inst.Spec.ServiceMonitorSelector,
		)
		if err != nil {
			level.Error(l).Log("msg", "unable to fetch service monitors", "err", err)
			return controller.Result{}, nil
		}
		smResources = append(smResources, smSelector)

		// TODO(rfratto): load in secrets from sMons

		// TODO(rfratto): pod monitors, probes

		prometheusInstances = append(prometheusInstances, operator.PrometheusInstance{
			Instance:        inst,
			ServiceMonitors: sMons,
		})
	}

	deployment := operator.Deployment{
		Agent:      &agent,
		Prometheis: prometheusInstances,
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
	r.eventPromInstances.Notify(req.NamespacedName, promResources)
	r.eventServiceMonitor.Notify(req.NamespacedName, smResources)
	r.eventPodMonitor.Notify(req.NamespacedName, pmResources)
	r.eventProbe.Notify(req.NamespacedName, probeResources)
	r.eventSecret.Notify(req.NamespacedName, secretResources)

	return controller.Result{}, nil
}

func (r *reconciler) prometheusInstances(
	ctx context.Context,
	ns string,
	nsLabelSelector *v1.LabelSelector,
	instanceLabelSelector *v1.LabelSelector,
) ([]*grafana_v1alpha1.PrometheusInstance, operator.ResourceSelector, error) {
	var resourceSelector operator.ResourceSelector

	var (
		// checkNamespace enforces looking for a single namespace. Defaults to the
		// namespace of the root resource, but changed to empty when a nsLabelSelector
		// is provided.
		checkNamespace    = ns
		namespaceSelector = labels.Everything()
	)

	if nsLabelSelector != nil {
		if sel, err := v1.LabelSelectorAsSelector(nsLabelSelector); err != nil {
			return nil, resourceSelector, fmt.Errorf("failed to create label selector for namespace: %w", err)
		} else {
			// We have a selector, so we want to get objects across all namespaces and filter later.
			namespaceSelector = sel
			checkNamespace = ""

			resourceSelector.NamespaceName = prom.NamespaceSelector{Any: true}
			resourceSelector.NamespaceLabels = namespaceSelector
		}
	} else {
		resourceSelector.NamespaceName = prom.NamespaceSelector{MatchNames: []string{ns}}
		resourceSelector.NamespaceLabels = labels.Everything()
	}

	var list grafana_v1alpha1.PrometheusInstanceList
	if sel, err := r.objectList(ctx, checkNamespace, instanceLabelSelector, &list); err != nil {
		return nil, resourceSelector, fmt.Errorf("error getting instances: %w", err)
	} else {
		resourceSelector.Labels = sel
	}

	// If we have a label selector, filter discovered items that don't match the namespace.
	if nsLabelSelector != nil {
		matches := make([]*grafana_v1alpha1.PrometheusInstance, 0, len(list.Items))

		for _, item := range list.Items {
			var ns core_v1.Namespace
			if err := r.Get(ctx, types.NamespacedName{Name: item.Namespace}, &ns); err != nil {
				return nil, resourceSelector, fmt.Errorf("error getting namespace: %w", err)
			}

			if namespaceSelector.Matches(labels.Set(ns.Labels)) {
				matches = append(matches, item)
			}
		}

		list.Items = matches
	}

	return list.Items, resourceSelector, nil
}

func (r *reconciler) serviceMonitors(
	ctx context.Context,
	ns string,
	nsLabelSelector *v1.LabelSelector,
	serviceMonitorLabelSelector *v1.LabelSelector,
) ([]*prom.ServiceMonitor, operator.ResourceSelector, error) {
	var resourceSelector operator.ResourceSelector

	var (
		// checkNamespace enforces looking for a single namespace. Defaults to the
		// namespace of the root resource, but changed to empty when a nsLabelSelector
		// is provided.
		checkNamespace    = ns
		namespaceSelector = labels.Everything()
	)

	if nsLabelSelector != nil {
		if sel, err := v1.LabelSelectorAsSelector(nsLabelSelector); err != nil {
			return nil, resourceSelector, fmt.Errorf("failed to create label selector for namespace: %w", err)
		} else {
			// We have a selector, so we want to get objects across all namespaces and filter later.
			namespaceSelector = sel
			checkNamespace = ""

			resourceSelector.NamespaceName = prom.NamespaceSelector{Any: true}
			resourceSelector.NamespaceLabels = namespaceSelector
		}
	} else {
		resourceSelector.NamespaceName = prom.NamespaceSelector{MatchNames: []string{ns}}
		resourceSelector.NamespaceLabels = labels.Everything()
	}

	var list prom.ServiceMonitorList
	if sel, err := r.objectList(ctx, checkNamespace, serviceMonitorLabelSelector, &list); err != nil {
		return nil, resourceSelector, fmt.Errorf("error getting instances: %w", err)
	} else {
		resourceSelector.Labels = sel
	}

	// If we have a label selector, filter discovered items that don't match the namespace.
	if nsLabelSelector != nil {
		matches := make([]*prom.ServiceMonitor, 0, len(list.Items))

		for _, item := range list.Items {
			var ns core_v1.Namespace
			if err := r.Get(ctx, types.NamespacedName{Name: item.Namespace}, &ns); err != nil {
				return nil, resourceSelector, fmt.Errorf("error getting namespace: %w", err)
			}

			if namespaceSelector.Matches(labels.Set(ns.Labels)) {
				matches = append(matches, item)
			}
		}

		list.Items = matches
	}

	return list.Items, resourceSelector, nil
}

func (r *reconciler) objectList(
	ctx context.Context,
	ns string,
	labelSelector *v1.LabelSelector,
	obj client.ObjectList,
) (labels.Selector, error) {
	var (
		selector = labels.Everything()
	)

	if labelSelector != nil {
		if sel, err := v1.LabelSelectorAsSelector(labelSelector); err != nil {
			return nil, fmt.Errorf("failed to create label selector: %w", err)
		} else {
			selector = sel
		}
	}

	return selector, r.List(ctx, obj, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     ns,
	})
}
