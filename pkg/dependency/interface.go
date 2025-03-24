package dependency

import (
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
)

type DependencyManager interface {
	SortRoles(rbg *workloadsv1.RoleBasedGroup) ([]*workloadsv1.RoleSpec, error)
	CheckDependencies(roles []*workloadsv1.RoleSpec) (bool, error)
}
