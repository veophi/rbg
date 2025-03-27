package dependency

import (
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

type DependencyManager interface {
	SortRoles(rbg *workloadsv1alpha1.RoleBasedGroup) ([]*workloadsv1alpha1.RoleSpec, error)
	// avoid dead circle
	CheckDependencies(rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (ready bool, err error)
}
