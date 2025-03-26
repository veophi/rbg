package dependency

import (
	"github.com/go-logr/logr"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
)

type defaultDepencyManager struct {
	log logr.Logger
}

func (sorter *defaultDepencyManager) SortRoles(rbg *workloadsv1.RoleBasedGroup) (roles []*workloadsv1.RoleSpec, err error) {
	// Implementation of topological sort based on dependencies
	// ... (omitted for brevity)
	// return rbg.Spec.Roles, nil
	roles = make([]*workloadsv1.RoleSpec, len(rbg.Spec.Roles))
	for i, role := range rbg.Spec.Roles {
		roles[i] = &role
	}
	return

}

func (sorter *defaultDepencyManager) CheckDependencies(rbg *workloadsv1.RoleBasedGroup, role *workloadsv1.RoleSpec) (ready bool, err error) {
	// Check if all dependencies are ready
	// ... (omitted for brevity)
	return true, nil
}

func NewtDepencyManager(log logr.Logger) DependencyManager {
	return &defaultDepencyManager{log: log}
}
