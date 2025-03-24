package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
)

type ResourceBuilder interface {
	buildStatefulSet(cr *workloadsv1.RoleBasedGroup, role workloadsv1.RoleSpec) (*appsv1.StatefulSet, error)
	buildService(cr *workloadsv1.RoleBasedGroup, role workloadsv1.RoleSpec) (*corev1.Service, error)
	ReconcileWorkloadByRole(cr *workloadsv1.RoleBasedGroup, role workloadsv1.RoleSpec) error
	ReconcileServiceByRole(cr *workloadsv1.RoleBasedGroup, role workloadsv1.RoleSpec) error
}
