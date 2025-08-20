package framework

import (
	"fmt"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

func expectRbgScalingAdapterEqual(rbgScalingAdapter *v1alpha1.RoleBasedGroupScalingAdapter, rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec, scale *autoscalingv1.Scale, expectedReplicas *int32) error {
	if rbgScalingAdapter == nil || rbgScalingAdapter.Spec.Replicas == nil {
		return fmt.Errorf("rbgScalingAdapter is nil")
	}

	targetRef := rbgScalingAdapter.Spec.ScaleTargetRef
	if targetRef == nil {
		return fmt.Errorf("ScaleTargetRef is nil")
	}

	if targetRef.Name != rbg.Name {
		return fmt.Errorf("ScaleTargetRef.Name %s does not match rbg.Name %s", targetRef.Name, rbg.Name)
	}

	if targetRef.Role != role.Name {
		return fmt.Errorf("ScaleTargetRef.Role %s does not match role.Name %s", targetRef.Role, role.Name)
	}

	if expectedReplicas != nil {
		if scale.Spec.Replicas != *expectedReplicas {
			return fmt.Errorf("Scale.Spec.Replicas %v does not match role.Replicas %v", scale.Spec.Replicas, *expectedReplicas)
		}

		if scale.Status.Replicas != *expectedReplicas {
			return fmt.Errorf("Scale.Status.Replicas %v does not match role.Replicas %v", scale.Status.Replicas, *expectedReplicas)
		}
	} else {
		expectedReplicas = role.Replicas //initial replicas
	}

	if *rbgScalingAdapter.Spec.Replicas != *expectedReplicas {
		return fmt.Errorf("ScalingAdapter.Spec.Replicas %v does not match expectedReplicas %v", *rbgScalingAdapter.Spec.Replicas, *expectedReplicas)
	}

	if *rbgScalingAdapter.Status.Replicas != *expectedReplicas {
		return fmt.Errorf("ScalingAdapter.Spec.Replicas %v does not match expectedReplicas %v", *rbgScalingAdapter.Status.Replicas, *expectedReplicas)
	}

	if *role.Replicas != *expectedReplicas {
		return fmt.Errorf("Role.Replicas %v does not match expectedReplicas %v", *role.Replicas, *expectedReplicas)
	}

	if rbgScalingAdapter.Status.Phase != v1alpha1.AdapterPhaseBound {
		return fmt.Errorf("ScalingAdapter.Status.Phase %s is not AdapterPhaseBound", rbgScalingAdapter.Status.Phase)
	}

	return nil
}
