package reconciler

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/utils"
)

type DeploymentReconciler struct {
	scheme *runtime.Scheme
	client client.Client
}

var _ WorkloadReconciler = &DeploymentReconciler{}

func NewDeploymentReconciler(scheme *runtime.Scheme, client client.Client) *DeploymentReconciler {
	return &DeploymentReconciler{scheme: scheme, client: client}
}

func (r *DeploymentReconciler) Reconciler(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) error {
	logger := log.FromContext(ctx)
	deployApplyConfig, err := r.constructDeployApplyConfiguration(ctx, rbg, role)
	if err != nil {
		logger.Error(err, "Failed to construct deploy apply configuration")
		return err
	}
	if err := utils.PatchObjectApplyConfiguration(ctx, r.client, deployApplyConfig, utils.PatchSpec); err != nil {
		logger.Error(err, "Failed to patch deploy apply configuration")
		return err
	}
	return nil
}

func (r *DeploymentReconciler) constructDeployApplyConfiguration(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
) (*appsapplyv1.DeploymentApplyConfiguration, error) {
	podReconciler := NewPodReconciler(r.scheme, r.client)
	podTemplateApplyConfiguration, err := podReconciler.ConstructPodTemplateSpecApplyConfiguration(ctx, rbg, role)
	if err != nil {
		return nil, err
	}

	// construct deployment apply configuration
	deployConfig := appsapplyv1.Deployment(rbg.GetWorkloadName(role), rbg.Namespace).
		WithSpec(appsapplyv1.DeploymentSpec().
			WithReplicas(*role.Replicas).
			WithTemplate(podTemplateApplyConfiguration).
			WithSelector(metaapplyv1.LabelSelector().
				WithMatchLabels(rbg.GetCommonLabelsFromRole(role)))).
		WithAnnotations(rbg.GetCommonAnnotationsFromRole(role)).
		WithLabels(rbg.GetCommonLabelsFromRole(role)).
		WithOwnerReferences(metaapplyv1.OwnerReference().
			WithAPIVersion(rbg.APIVersion).
			WithKind(rbg.Kind).
			WithName(rbg.Name).
			WithUID(rbg.GetUID()).
			WithBlockOwnerDeletion(true).
			WithController(true),
		)
	return deployConfig, nil

}

func (r *DeploymentReconciler) ConstructRoleStatus(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
) (workloadsv1alpha1.RoleStatus, error) {
	deploy := &appsv1.Deployment{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, deploy); err != nil {
		return workloadsv1alpha1.RoleStatus{}, err
	}

	return workloadsv1alpha1.RoleStatus{
		Name:          role.Name,
		Replicas:      *deploy.Spec.Replicas,
		ReadyReplicas: deploy.Status.ReadyReplicas,
	}, nil
}
func (r *DeploymentReconciler) GetWorkloadType() string {
	return "apps/v1/Deployment"
}

func (r *DeploymentReconciler) CheckWorkloadReady(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (bool, error) {
	deploy := &appsv1.Deployment{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, deploy); err != nil {
		return false, err
	}
	return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas, nil
}
