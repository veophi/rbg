package utils

import (
	appsv1 "k8s.io/api/apps/v1"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

func SortRolesByDependencies(rbg *workloadsv1alpha1.RoleBasedGroup) (roles []*workloadsv1alpha1.RoleSpec, err error) {
	// Implementation of topological sort based on dependencies
	// ... (omitted for brevity)
	// return rbg.Spec.Roles, nil
	roles = make([]*workloadsv1alpha1.RoleSpec, len(rbg.Spec.Roles))
	for i, role := range rbg.Spec.Roles {
		roles[i] = &role
	}
	return

}

func CheckDependencies(rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (bool, error) {
	// Check if all dependencies are ready
	// ... (omitted for brevity)
	return true, nil
}

func UpdateRoleReplicas(
	cr *workloadsv1alpha1.RoleBasedGroup,
	roleName string,
	sts *appsv1.StatefulSet,
) bool {
	updateStatus := false
	replicas := int32(0)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	// 查找或创建角色状态记录
	index := -1
	for i, s := range cr.Status.RoleStatuses {
		if s.Name == roleName {
			index = i
			break
		}
	}

	if index == -1 {
		cr.Status.RoleStatuses = append(cr.Status.RoleStatuses, workloadsv1alpha1.RoleStatus{
			Name:          roleName,
			Replicas:      replicas,
			ReadyReplicas: sts.Status.ReadyReplicas,
		})
		updateStatus = true
	} else {
		// cr.Status.RoleStatuses[index].Replicas = replicas
		// cr.Status.RoleStatuses[index].ReadyReplicas = sts.Status.ReadyReplicas
		if cr.Status.RoleStatuses[index].Replicas != replicas {
			cr.Status.RoleStatuses[index].Replicas = replicas
			updateStatus = true
		}

		if cr.Status.RoleStatuses[index].ReadyReplicas != sts.Status.ReadyReplicas {
			cr.Status.RoleStatuses[index].ReadyReplicas = sts.Status.ReadyReplicas
			updateStatus = true
		}

	}

	return updateStatus
}
