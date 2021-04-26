package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/grafana/agent/pkg/build"
	"github.com/grafana/agent/pkg/operator/assets"
	"github.com/grafana/agent/pkg/operator/config"
	"github.com/pkg/errors"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/validation"
)

func generateStatefulSet(d config.Deployment, secrets assets.SecretStore) (*apps_v1.StatefulSet, error) {
	// TODO(rfratto): stuff that happens in prometheus-operator:
	// - default values for port name, replicas, retention, requests
	// - make stateful set spec
	// - add extra kubectl annotations
	// - generate statefulset
	// - add annotations to statefulset
	// - add image pull secrets
	// - add storage spec to default the storage, else create PVC template
	// - combine volumes of template spec with volumes of SS spec

	spec, err := generateStatefulSetSpec(d, secrets)
	if err != nil {
		return nil, err
	}

	boolTrue := true

	ss := &apps_v1.StatefulSet{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        d.Agent.Name,
			Namespace:   d.Agent.Namespace,
			Labels:      spec.Template.Labels,
			Annotations: spec.Template.Annotations,
			OwnerReferences: []meta_v1.OwnerReference{{
				APIVersion:         d.Agent.APIVersion,
				Kind:               d.Agent.Kind,
				BlockOwnerDeletion: &boolTrue,
				Controller:         &boolTrue,
				Name:               d.Agent.Name,
				UID:                d.Agent.UID,
			}},
		},
		Spec: *spec,
	}

	// TODO(rfratto): input hash annotation?

	if len(d.Agent.Spec.ImagePullSecrets) > 0 {
		ss.Spec.Template.Spec.ImagePullSecrets = d.Agent.Spec.ImagePullSecrets
	}

	storageSpec := d.Agent.Spec.Storage
	if storageSpec == nil {
		ss.Spec.Template.Spec.Volumes = append(ss.Spec.Template.Spec.Volumes, v1.Volume{
			Name: fmt.Sprintf("%s-wal", d.Agent.Name),
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	} else if storageSpec.EmptyDir != nil {
		emptyDir := storageSpec.EmptyDir
		ss.Spec.Template.Spec.Volumes = append(ss.Spec.Template.Spec.Volumes, v1.Volume{
			Name: fmt.Sprintf("%s-wal", d.Agent.Name),
			VolumeSource: v1.VolumeSource{
				EmptyDir: emptyDir,
			},
		})
	} else {
		// TODO(rfratto): handle creating PVC template
		return nil, fmt.Errorf("cannot create PVC template yet")
	}

	return ss, nil
}

func generateStatefulSetSpec(d config.Deployment, secrets assets.SecretStore) (*apps_v1.StatefulSetSpec, error) {
	terminationGracePeriodSeconds := int64(4800)

	var (
		portName = defaultString(d.Agent.Spec.PortName, "agent-metrics")
	)

	agentArgs := []string{
		"-config.file=/var/lib/grafana-agent/config/agent.yml",
		"-config.expand-env=true",
	}

	// TODO(rfratto): ListenLocal on the Agent spec, which would leave this list
	// as empty.
	ports := []v1.ContainerPort{{
		Name:          portName,
		ContainerPort: 8080,
		Protocol:      v1.ProtocolTCP,
	}}

	volumes := []v1.Volume{
		{
			Name: "config",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-config", d.Agent.Name),
				},
			},
		},
		{
			Name: "secrets",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-secrets", d.Agent.Name),
				},
			},
		},
	}

	walVolumeName := fmt.Sprintf("%s-wal", d.Agent.Name)
	if d.Agent.Spec.Storage != nil {
		if d.Agent.Spec.Storage.VolumeClaimTemplate.Name != "" {
			walVolumeName = d.Agent.Spec.Storage.VolumeClaimTemplate.Name
		}
	}

	volumeMounts := []v1.VolumeMount{
		{
			Name:      "config",
			ReadOnly:  true,
			MountPath: "/var/lib/grafana-agent/config",
		},
		{
			Name:      walVolumeName,
			ReadOnly:  false,
			MountPath: "/var/lib/grafana-agent/data",
		},
		{
			Name:      "secrets",
			ReadOnly:  true,
			MountPath: "/var/lib/grafana-agent/secrets",
		},
	}

	for _, s := range d.Agent.Spec.Secrets {
		volumes = append(volumes, v1.Volume{
			Name: sanitizeVolumeName("secret-" + s),
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{SecretName: s},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      sanitizeVolumeName("secret-" + s),
			ReadOnly:  true,
			MountPath: "/var/lib/grafana-agent/secrets",
		})
	}

	for _, c := range d.Agent.Spec.ConfigMaps {
		volumes = append(volumes, v1.Volume{
			Name: sanitizeVolumeName("configmap-" + c),
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: c},
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      sanitizeVolumeName("configmap-" + c),
			ReadOnly:  true,
			MountPath: "/var/lib/grafana-agent/configmaps",
		})
	}

	// TODO(rfratto): Readiness probe?

	podAnnotations := map[string]string{}
	podLabels := map[string]string{}
	podSelectorLabels := map[string]string{
		"app.kubernetes.io/name":           "prometheus",
		"app.kubernetes.io/version":        build.Version,
		"app.kubernetes.io/managed-by":     "grafana-agent-operator",
		"app.kubernetes.io/instance":       d.Agent.Name,
		"grafana-agent":                    d.Agent.Name,
		"operator.agent.grafana.com/shard": "0", // TODO(rfratto): fix
		"operator.agent.grafana.com/name":  d.Agent.Name,
	}
	if d.Agent.Spec.PodMetadata != nil {
		if d.Agent.Spec.PodMetadata.Labels != nil {
			for k, v := range d.Agent.Spec.PodMetadata.Labels {
				podLabels[k] = v
			}
		}
		if d.Agent.Spec.PodMetadata.Annotations != nil {
			for k, v := range d.Agent.Spec.PodMetadata.Annotations {
				podAnnotations[k] = v
			}
		}
	}
	for k, v := range podSelectorLabels {
		podLabels[k] = v
	}

	podAnnotations["kubectl.kubernetes.io/default-container"] = "grafana-agent"

	// TODO(rfratto): final selector labels, labels based on operator config

	operatorContainers := append([]v1.Container{
		{
			Name:         "grafana-agent",
			Image:        "grafana/agent:v0.14.0-rc.2", // TODO(rfratto): grab from config
			Ports:        ports,
			Args:         agentArgs,
			VolumeMounts: volumeMounts,
			Env: []v1.EnvVar{{
				Name:  "SHARD",
				Value: "0", // TODO(rfratto): fix to be correct
			}},
			// TODO(rfratto): readiness probe
			Resources:                d.Agent.Spec.Resources,
			TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
		},
	})

	containers, err := mergePatchContainers(operatorContainers, d.Agent.Spec.Containers)
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec: %w", err)
	}

	return &apps_v1.StatefulSetSpec{
		ServiceName:         "grafana-agent-operated",
		Replicas:            d.Agent.Spec.Prometheus.Replicas,
		PodManagementPolicy: apps_v1.ParallelPodManagement,
		UpdateStrategy: apps_v1.StatefulSetUpdateStrategy{
			Type: apps_v1.RollingUpdateStatefulSetStrategyType,
		},
		Selector: &meta_v1.LabelSelector{
			MatchLabels: podSelectorLabels, // TODO(rfratto): finalSelectorLabels
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: meta_v1.ObjectMeta{
				Labels:      podLabels,
				Annotations: podAnnotations,
			},
			Spec: v1.PodSpec{
				Containers:                    containers,
				InitContainers:                d.Agent.Spec.InitContainers,
				SecurityContext:               d.Agent.Spec.SecurityContext,
				ServiceAccountName:            d.Agent.Spec.ServiceAccountName,
				NodeSelector:                  d.Agent.Spec.NodeSelector,
				PriorityClassName:             d.Agent.Spec.PriorityClassName,
				TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				Volumes:                       volumes,
				Tolerations:                   d.Agent.Spec.Tolerations,
				Affinity:                      d.Agent.Spec.Affinity,
				TopologySpreadConstraints:     d.Agent.Spec.TopologySpreadConstraints,
			},
		},
	}, nil
}

func defaultString(val string, def string) string {
	if val == "" {
		return def
	}
	return val
}

var invalidDNS1123Characters = regexp.MustCompile("[^-a-z0-9]+")

func sanitizeVolumeName(name string) string {
	name = strings.ToLower(name)
	name = invalidDNS1123Characters.ReplaceAllString(name, "-")
	if len(name) > validation.DNS1123LabelMaxLength {
		name = name[0:validation.DNS1123LabelMaxLength]
	}
	return strings.Trim(name, "-")
}

// mergePatchContainers adds patches to base using a strategic merge patch and
// iterating by container name, failing on the first error
func mergePatchContainers(base, patches []v1.Container) ([]v1.Container, error) {
	var out []v1.Container

	// map of containers that still need to be patched by name
	containersToPatch := make(map[string]v1.Container)
	for _, c := range patches {
		containersToPatch[c.Name] = c
	}

	for _, container := range base {
		// If we have a patch result, iterate over each container and try and calculate the patch
		if patchContainer, ok := containersToPatch[container.Name]; ok {
			// Get the json for the container and the patch
			containerBytes, err := json.Marshal(container)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to marshal json for container %s", container.Name))
			}
			patchBytes, err := json.Marshal(patchContainer)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to marshal json for patch container %s", container.Name))
			}

			// Calculate the patch result
			jsonResult, err := strategicpatch.StrategicMergePatch(containerBytes, patchBytes, v1.Container{})
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to generate merge patch for %s", container.Name))
			}
			var patchResult v1.Container
			if err := json.Unmarshal(jsonResult, &patchResult); err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to unmarshal merged container %s", container.Name))
			}

			// Add the patch result and remove the corresponding key from the to do list
			out = append(out, patchResult)
			delete(containersToPatch, container.Name)
		} else {
			// This container didn't need to be patched
			out = append(out, container)
		}
	}

	// Iterate over the patches and add all the containers that were not previously part of a patch result
	for _, container := range patches {
		if _, ok := containersToPatch[container.Name]; ok {
			out = append(out, container)
		}
	}

	return out, nil
}
