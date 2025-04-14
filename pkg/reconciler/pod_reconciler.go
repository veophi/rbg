package reconciler

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	coreapplyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/discovery"
)

type PodReconciler struct {
	scheme *runtime.Scheme
	client client.Client
}

func NewPodReconciler(scheme *runtime.Scheme, client client.Client) *PodReconciler {
	return &PodReconciler{
		scheme: scheme,
		client: client,
	}
}

func (r *PodReconciler) ConstructPodTemplateSpecApplyConfiguration(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
) (*coreapplyv1.PodTemplateSpecApplyConfiguration, error) {
	podTemplateSpec := *role.Template.DeepCopy()

	injector := discovery.NewDefaultInjector(r.scheme, r.client)
	if err := injector.InjectConfig(ctx, &podTemplateSpec, rbg, role); err != nil {
		return nil, fmt.Errorf("failed to inject config: %w", err)
	}
	// sidecar也需要rbg相关的env，先注入sidecar
	if err := injector.InjectSidecar(ctx, &podTemplateSpec, rbg, role); err != nil {
		return nil, fmt.Errorf("failed to inject sidecar: %w", err)
	}
	if err := injector.InjectEnv(ctx, &podTemplateSpec, rbg, role); err != nil {
		return nil, fmt.Errorf("failed to inject env vars: %w", err)
	}

	// construct pod template spec configuration
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&podTemplateSpec)
	if err != nil {
		return nil, err
	}
	var podTemplateApplyConfiguration *coreapplyv1.PodTemplateSpecApplyConfiguration
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj, &podTemplateApplyConfiguration)
	if err != nil {
		return nil, err
	}
	podTemplateApplyConfiguration.WithLabels(rbg.GetCommonLabelsFromRole(role))
	podTemplateApplyConfiguration.WithAnnotations(rbg.GetCommonAnnotationsFromRole(role))

	return podTemplateApplyConfiguration, nil
}
