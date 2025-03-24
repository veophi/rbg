package utils

import (
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
)

func SortRolesByDependencies(rbg *workloadsv1.RoleBasedGroup) ([]workloadsv1.RoleSpec, error) {
	// Implementation of topological sort based on dependencies
	// ... (omitted for brevity)
	return rbg.Spec.Roles, nil
}

func CheckDependencies(rbg *workloadsv1.RoleBasedGroup, role workloadsv1.RoleSpec) (bool, error) {
	// Check if all dependencies are ready
	// ... (omitted for brevity)
	return true, nil
}
