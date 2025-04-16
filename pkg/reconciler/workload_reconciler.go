package reconciler

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

type WorkloadReconciler interface {
	Reconciler(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) error
	ConstructRoleStatus(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (workloadsv1alpha1.RoleStatus, error)
	GetWorkloadType() string
	CheckWorkloadReady(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (bool, error)
}

func NewWorkloadReconciler(apiVersion, kind string, scheme *runtime.Scheme, client client.Client) (WorkloadReconciler, error) {
	switch {
	case apiVersion == "apps/v1" && kind == "Deployment":
		return NewDeploymentReconciler(scheme, client), nil
	case apiVersion == "apps/v1" && kind == "StatefulSet":
		return NewStatefulSetReconciler(scheme, client), nil
	default:
		return nil, fmt.Errorf("unsupported workload type: %s/%s", apiVersion, kind)
	}
}
