package informers

import (
	"time"

	extinformers "github.com/grafana/agent/pkg/operator/client/informers/externalversions"
	monitoring "github.com/grafana/agent/pkg/operator/client/versioned"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/listwatch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

// NewMonitoringInformerFactories creates factories for monitoring resources
// for the given allowed, and denied namespaces these parameters being mutually
// exclusive.  monitoringClient, defaultResync, and tweakListOptions are being
// passed to the underlying informer factory.
func NewMonitoringInformerFactories(
	allowNamespaces, denyNamespaces map[string]struct{},
	monitoringClient monitoring.Interface,
	defaultResync time.Duration,
	tweakListOptions func(*metav1.ListOptions),
) informers.FactoriesForNamespaces {
	tweaks, namespaces := newInformerOptions(
		allowNamespaces, denyNamespaces, tweakListOptions,
	)

	opts := []extinformers.SharedInformerOption{extinformers.WithTweakListOptions(tweaks)}

	ret := monitoringInformersForNamespaces{}
	for _, namespace := range namespaces {
		opts = append(opts, extinformers.WithNamespace(namespace))
		ret[namespace] = extinformers.NewSharedInformerFactoryWithOptions(monitoringClient, defaultResync, opts...)
	}

	return ret
}

type monitoringInformersForNamespaces map[string]extinformers.SharedInformerFactory

func (i monitoringInformersForNamespaces) Namespaces() sets.String {
	return sets.StringKeySet(i)
}

func (i monitoringInformersForNamespaces) ForResource(namespace string, resource schema.GroupVersionResource) (informers.InformLister, error) {
	return i[namespace].ForResource(resource)
}

// newInformerOptions returns a list option tweak function and a list of namespaces
// based on the given allowed and denied namespaces.
//
// If allowedNamespaces contains one only entry equal to k8s.io/apimachinery/pkg/apis/meta/v1.NamespaceAll
// then it returns it and a tweak function filtering denied namespaces using a field selector.
//
// Else, denied namespaces are ignored and just the set of allowed namespaces is returned.
func newInformerOptions(allowedNamespaces, deniedNamespaces map[string]struct{}, tweaks func(*v1.ListOptions)) (func(*v1.ListOptions), []string) {
	if tweaks == nil {
		tweaks = func(*v1.ListOptions) {} // nop
	}

	var namespaces []string

	if listwatch.IsAllNamespaces(allowedNamespaces) {
		return func(options *v1.ListOptions) {
			tweaks(options)
			listwatch.DenyTweak(options, "metadata.namespace", deniedNamespaces)
		}, []string{v1.NamespaceAll}
	}

	for ns := range allowedNamespaces {
		namespaces = append(namespaces, ns)
	}

	return tweaks, namespaces
}
