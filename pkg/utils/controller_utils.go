package utils

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	wlReplicas *int32,
	wlReadyReplicas int32,
) bool {
	updateStatus := false
	replicas := int32(0)
	if wlReplicas != nil {
		replicas = *wlReplicas
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
			ReadyReplicas: wlReadyReplicas,
		})
		updateStatus = true
	} else {
		// cr.Status.RoleStatuses[index].Replicas = replicas
		// cr.Status.RoleStatuses[index].ReadyReplicas = sts.Status.ReadyReplicas
		if cr.Status.RoleStatuses[index].Replicas != replicas {
			cr.Status.RoleStatuses[index].Replicas = replicas
			updateStatus = true
		}

		if cr.Status.RoleStatuses[index].ReadyReplicas != wlReadyReplicas {
			cr.Status.RoleStatuses[index].ReadyReplicas = wlReadyReplicas
			updateStatus = true
		}

	}

	return updateStatus
}

func UpdateRbgStatus(
	ctx context.Context,
	client client.Client,
	oldStatus *workloadsv1alpha1.RoleBasedGroupStatus,
	newRbg *workloadsv1alpha1.RoleBasedGroup,
) error {
	logger := log.FromContext(ctx)
	if reflect.DeepEqual(oldStatus, newRbg.Status) {
		logger.V(1).Info("No need to update for old status  and new status , because it's deepequal", "oldStatus", oldStatus, "newStatus", newRbg.Status)
		return nil
	}

	if err := client.Status().Update(ctx, newRbg); err != nil {
		if !apierrors.IsConflict(err) {
			logger.Error(err, "Updating LeaderWorkerSet status and/or condition.")
		}
		return err
	}
	return nil
}
