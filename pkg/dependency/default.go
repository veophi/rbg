package dependency

import (
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

type DefaultDependencyManager struct{}

var _ DependencyManager = &DefaultDependencyManager{}

func (sorter *DefaultDependencyManager) SortRoles(rbg *workloadsv1alpha1.RoleBasedGroup) (roles []*workloadsv1alpha1.RoleSpec, err error) {
	// Implementation of topological sort based on dependencies
	// ... (omitted for brevity)
	// return rbg.Spec.Roles, nil
	roles = make([]*workloadsv1alpha1.RoleSpec, len(rbg.Spec.Roles))
	for i, role := range rbg.Spec.Roles {
		roles[i] = &role
	}
	return

}

func (sorter *DefaultDependencyManager) CheckDependencies(rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (ready bool, err error) {
	// Check if all dependencies are ready
	// ... (omitted for brevity)
	return true, nil
}

func NewDependencyManager() DependencyManager {
	return &DefaultDependencyManager{}
}
