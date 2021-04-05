package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Various keys for the CRDs.
const (
	Version = "v1"

	GrafanaAgentsKind   = "GrafanaAgent"
	GrafanaAgentName    = "grafana-agents"
	GrafanaAgentKindKey = "grafana-agent"
)

// GrafanaAgent defines a deployment of Grafana Agents.
// +genclient
// +k8s:openapi-gen=true
// +kubebuilder:resource:path="grafana-agents"
// +kubebuilder:resource:singular="grafana-agent"
// +kubebuilder:resource:categories="agent-operator"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="The version of Grafana Agent"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas",description="The desired replicas number of Grafana Agents"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type GrafanaAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the Grafana Agent cluster.
	Spec GrafanaAgentSpec `json:"spec"`
	// Most recent observed status of the Grafana Agent cluster. Read-only. Not
	// included when requesting from the apiserver, only from the Grafana Agent
	// Operator API itself.
	Status *GrafanaAgentStatus `json:"status,omitempty"`
}

// DeepCopyObject implements the runtime.Object interface.
func (o *GrafanaAgent) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

// GrafanaAgentList is a list of GrafanaAgents.
// +k8s:openapi-gen=true
type GrafanaAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of GrafanaAgents.
	Items []*GrafanaAgent `json:"items"`
}

// DeepCopyObject implements the runtime.Object interface.
func (o *GrafanaAgentList) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

// GrafanaAgentSpec is a specification of the desired behavior of the Grafana
// Agent cluster.
// +k8s:openapi-gen=true
type GrafanaAgentSpec struct {
	// TODO(rfratto): fill out
}

// GrafanaAgentStatus is the most recent observed status of the Grafana Agent
// cluster. Read-only. Not included when requesting from the apiserver, only
// from the Grafana Agent operator API itself.
// +k8s:openapi-gen=true
type GrafanaAgentStatus struct {
	// TODO(rfratto): fill out
}
