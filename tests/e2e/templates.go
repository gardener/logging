// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/json"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func newBackendStatefulSet(namespace string, name string, image string) *appsv1.StatefulSet {

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{"app.kubernetes.io/name": "vali"}},
		Spec: appsv1.StatefulSetSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "vali"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "vali"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "vali",
							Image: image,
							Ports: []corev1.ContainerPort{{
								ContainerPort: 3100,
							}},
							VolumeMounts: []corev1.VolumeMount{{Name: "vali", MountPath: "/var/log/vali"}},
						},
					},
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
					Volumes: []corev1.Volume{{
						Name:         "vali",
						VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					},
					},
				},
			},
		},
	}
}

func newBackendService(namespace string, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "vali"},
			Ports: []corev1.ServicePort{{
				Name: "vali",
				Port: 3100,
			}},
		},
	}
}

func newFluentBitDaemonSet(namespace string, name string, image string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{"app.kubernetes.io/name": "fluent-bit"}},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "fluent-bit"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "fluent-bit"}},
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					Containers: []corev1.Container{
						{
							Name:  "fluent-bit",
							Image: image,
							Ports: []corev1.ContainerPort{{
								ContainerPort: 2020,
							}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "fluent-bit-config",
									MountPath: "/fluent-bit/config/fluent-bit.conf",
									SubPath:   "fluent-bit.conf",
								},
								{
									Name:      "fluent-bit-config",
									MountPath: "/fluent-bit/config/add_tag_to_record.lua",
									SubPath:   "add_tag_to_record.lua",
								},
								{
									Name:      "var-log",
									MountPath: "/var/log",
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: ptr.To(int64(0)),
					},
					Volumes: []corev1.Volume{
						{
							Name: "fluent-bit-config",
							VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.
								ConfigMapVolumeSource{LocalObjectReference: corev1.
								LocalObjectReference{Name: DaemonSetName + "-config"},
								Optional: ptr.To(false)}},
						},
						{
							Name:         "var-log",
							VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/log"}},
						},
					},
				},
			},
		},
	}
}

func newFluentBitConfigMap(namespace string, data string, lua string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: DaemonSetName + "-config", Namespace: namespace},
		Data:       map[string]string{"fluent-bit.conf": data, "add_tag_to_record.lua": lua},
	}
}

func newFluentBitRBAC(namespace string, name string) (*v1.ClusterRole, *v1.ClusterRoleBinding) {
	clusterRole := &v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{"extensions.gardener.cloud"},
				Resources: []string{"clusters"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	clusterRoleBinding := &v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
	}

	return clusterRole, clusterRoleBinding
}

func newServiceAccount(namespace string, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
}

func newLoggerPod(namespace string, name string) *corev1.Pod {

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{"run": "logs-generator"}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "logs-generator",
					Image: "nickytd/log-generator:latest", // TODO: pre-load the image in the cluster
					Env: []corev1.EnvVar{
						{Name: "LOGS_COUNT", Value: "1000"},
						{Name: "LOGS_WAIT", Value: "10ms"},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Tolerations:   []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
		},
	}
}

func newExtensionCluster(name string, state string) *extensionsv1alpha1.Cluster {
	shoot := &gardencorev1beta1.Shoot{
		Spec: gardencorev1beta1.ShootSpec{
			Hibernation: &gardencorev1beta1.Hibernation{
				Enabled: ptr.To(false),
			},
			Purpose: (*gardencorev1beta1.ShootPurpose)(ptr.To("evaluation")),
		},
	}

	switch state {
	case "create":
		shoot.Status.LastOperation = &gardencorev1beta1.LastOperation{
			Type:  gardencorev1beta1.LastOperationTypeCreate,
			State: gardencorev1beta1.LastOperationStateProcessing,
		}
	case "deletion":
		shoot.DeletionTimestamp = &metav1.Time{}
	case "hibernating":
		shoot.Spec.Hibernation.Enabled = ptr.To(true)
		shoot.Status.IsHibernated = false
	case "hibernated":
		shoot.Spec.Hibernation.Enabled = ptr.To(true)
		shoot.Status.IsHibernated = true
	case "wailing":
		shoot.Spec.Hibernation.Enabled = ptr.To(false)
		shoot.Status.IsHibernated = true
	case "ready":
		shoot.Status.LastOperation = &gardencorev1beta1.LastOperation{
			Type:  gardencorev1beta1.LastOperationTypeReconcile,
			State: gardencorev1beta1.LastOperationStateSucceeded,
		}
	}

	return &extensionsv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "extensions.gardener.cloud/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			Shoot: runtime.RawExtension{
				Raw: encode(shoot),
			},
			CloudProfile: runtime.RawExtension{
				Raw: encode(&gardencorev1beta1.CloudProfile{}),
			},
			Seed: runtime.RawExtension{
				Raw: encode(&gardencorev1beta1.Seed{}),
			},
		},
	}
}

func newEventLoggerRBAC(namespace string, name string) (*v1.Role, *v1.RoleBinding) {
	role := &v1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	roleBinding := &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
	}
	return role, roleBinding
}

func newEventLoggerDeployment(namespace string, name string, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{"apps.kubernetes.io/name": "event-logger"}},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "event-logger"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "event-logger"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					Containers: []corev1.Container{
						{
							Name:    "event-logger",
							Image:   image,
							Command: []string{"./event-logger", "--seed-event-namespaces=" + namespace},
						},
					},
					Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
				},
			},
		},
	}
}

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}
