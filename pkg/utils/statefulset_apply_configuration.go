package utils

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	coreapplyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

func ConstructStatefulsetApplyConfigurationByRole(rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (apply *appsapplyv1.StatefulSetApplyConfiguration, err error) {
	podTemplateSpec := role.Template

	// Inject environment variables
	envVars := injectDiscoveryConfigToEnv(role)
	for i := range podTemplateSpec.Spec.Containers {
		podTemplateSpec.Spec.Containers[i].Env = append(
			podTemplateSpec.Spec.Containers[i].Env, envVars...)
	}

	// construct pod template spec configuration
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&podTemplateSpec)
	if err != nil {
		return nil, err
	}
	var podTemplateApplyConfiguration coreapplyv1.PodTemplateSpecApplyConfiguration
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj, &podTemplateApplyConfiguration)
	if err != nil {
		return nil, err
	}

	// construct statefulset apply configuration
	statefulSetConfig := appsapplyv1.StatefulSet(role.Name, rbg.Namespace).
		WithSpec(appsapplyv1.StatefulSetSpec().
			WithServiceName(role.Name).
			WithReplicas(*role.Replicas).
			WithPodManagementPolicy(appsv1.ParallelPodManagement).
			WithTemplate(&podTemplateApplyConfiguration)).
		WithOwnerReferences(metaapplyv1.OwnerReference().
			WithAPIVersion(rbg.APIVersion).
			WithKind(rbg.Kind).
			WithName(rbg.Name).
			WithUID(rbg.GetUID()).
			WithBlockOwnerDeletion(true).
			WithController(true)).
		WithLabels(rbg.GetCommonLabelsFromRole(role)).
		WithAnnotations(rbg.GetCommonAnnotationsFromRole(role))
	return statefulSetConfig, nil

}
