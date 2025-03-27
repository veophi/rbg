package utils

import (
	corev1 "k8s.io/api/core/v1"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

func injectDiscoveryConfigToEnv(role *workloadsv1alpha1.RoleSpec) (envVars []corev1.EnvVar) {
	// Inject environment variables for service discovery
	envVars = []corev1.EnvVar{
		{
			Name:  "ROLE_NAME",
			Value: role.Name,
		},
		{
			Name: "ROLE_INDEX",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.annotations['apps.kubernetes.io/pod-index']",
				},
			},
		},
	}

	return
}
