package utils

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

func ConstructStatefulsetByRole(rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec, scheme *runtime.Scheme) (sts *appsv1.StatefulSet, err error) {
	// Generate StatefulSet name
	stsName := fmt.Sprintf("%s-%s", rbg.Name, role.Name)

	// Create or update StatefulSet
	sts = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        stsName,
			Namespace:   rbg.Namespace,
			Labels:      rbg.GetCommonLabelsFromRole(role),
			Annotations: rbg.GetCommonAnnotationsFromRole(role),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: role.Replicas,
			Template: role.Template,
		},
	}

	// Set controller reference
	err = ctrl.SetControllerReference(rbg, sts, scheme)
	if err != nil {
		return
	}

	// Inject environment variables
	envVars := injectDiscoveryConfigToEnv(role)
	// Add environment variables to all containers
	for i := range sts.Spec.Template.Spec.Containers {
		sts.Spec.Template.Spec.Containers[i].Env = append(
			sts.Spec.Template.Spec.Containers[i].Env,
			envVars...,
		)
	}

	return
}

func ReconcileStatefulSet(ctx context.Context, k8sClient client.Client, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec, scheme *runtime.Scheme) (err error) {
	sts, err := ConstructStatefulsetApplyConfigurationByRole(rbg, role)
	if err != nil {
		return
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(sts)
	if err != nil {
		return err
	}
	workerStatefulSet := &unstructured.Unstructured{
		Object: obj,
	}

	var workerSts appsv1.StatefulSet
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: rbg.Namespace}, &workerSts); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		if err = k8sClient.Create(ctx, workerStatefulSet); err != nil {
			return client.IgnoreAlreadyExists(err)
		}
	}

	return nil
}
