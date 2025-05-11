package reconciler

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

type WorkloadReconciler interface {
	Reconciler(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) error
	ConstructRoleStatus(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (workloadsv1alpha1.RoleStatus, bool, error)
	GetWorkloadType() string
	CheckWorkloadReady(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (bool, error)
	CleanupOrphanedWorkloads(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup) error
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

// WorkloadEqual determines whether the workload needs reconciliation
func WorkloadEqual(obj1, obj2 interface{}) (bool, error) {
	switch o1 := obj1.(type) {
	case *appsv1.Deployment:
		if o2, ok := obj2.(*appsv1.Deployment); ok {
			// check spec
			if equal, err := SemanticallyEqualDeployment(o1, o2); !equal {
				return false, fmt.Errorf("deploy not equal, error: %s", err.Error())
			}
			// check status
			if o1.Status.ReadyReplicas != o2.Status.ReadyReplicas {
				return false, fmt.Errorf("ReadyReplicas not equal, old: %d, new: %d", o1.Status.ReadyReplicas, o2.Status.ReadyReplicas)
			}

			return true, nil
		}
	case *appsv1.StatefulSet:
		if o2, ok := obj2.(*appsv1.StatefulSet); ok {
			// check spec
			if equal, err := SemanticallyEqualStatefulSet(o1, o2); !equal {
				return false, fmt.Errorf("sts not equal, error: %s", err.Error())
			}
			// check status
			if o1.Status.ReadyReplicas != o2.Status.ReadyReplicas {
				return false, fmt.Errorf("ReadyReplicas not equal, old: %d, new: %d", o1.Status.ReadyReplicas, o2.Status.ReadyReplicas)
			}

			return true, nil
		}
	}

	return false, fmt.Errorf("not support workload: %v", reflect.TypeOf(obj1))
}
