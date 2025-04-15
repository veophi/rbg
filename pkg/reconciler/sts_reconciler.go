package reconciler

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	coreapplyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/utils"
)

type StatefulSetReconciler struct {
	scheme *runtime.Scheme
	client client.Client
}

var _ WorkloadReconciler = &StatefulSetReconciler{}

func NewStatefulSetReconciler(scheme *runtime.Scheme, client client.Client) *StatefulSetReconciler {
	return &StatefulSetReconciler{
		scheme: scheme,
		client: client,
	}
}

func (r *StatefulSetReconciler) Reconciler(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) error {
	logger := log.FromContext(ctx)
	logger.Info("start to reconciler sts workload")

	stsApplyConfig, err := r.constructStatefulSetApplyConfiguration(ctx, rbg, role)
	if err != nil {
		logger.Error(err, "Failed to construct statefulset apply configuration")
		return err
	}
	if err := utils.PatchObjectApplyConfiguration(ctx, r.client, stsApplyConfig, utils.PatchSpec); err != nil {
		logger.Error(err, "Failed to patch statefulset apply configuration")
		return err
	}

	svcApplyConfig, err := r.constructServiceApplyConfiguration(ctx, rbg, role)
	if err != nil {
		return err
	}

	return utils.PatchObjectApplyConfiguration(ctx, r.client, svcApplyConfig, utils.PatchSpec)
}

func (r *StatefulSetReconciler) constructStatefulSetApplyConfiguration(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
) (*appsapplyv1.StatefulSetApplyConfiguration, error) {

	podReconciler := NewPodReconciler(r.scheme, r.client)
	podTemplateApplyConfiguration, err := podReconciler.ConstructPodTemplateSpecApplyConfiguration(ctx, rbg, role)
	if err != nil {
		return nil, err
	}

	// construct statefulset apply configuration
	statefulSetConfig := appsapplyv1.StatefulSet(rbg.GetWorkloadName(role), rbg.Namespace).
		WithSpec(appsapplyv1.StatefulSetSpec().
			WithServiceName(rbg.GetWorkloadName(role)).
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
	return statefulSetConfig, nil
}

func (r *StatefulSetReconciler) constructServiceApplyConfiguration(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
) (*coreapplyv1.ServiceApplyConfiguration, error) {
	selectMap := map[string]string{
		workloadsv1alpha1.SetNameLabelKey: rbg.Name,
		workloadsv1alpha1.SetRoleLabelKey: role.Name,
	}
	serviceConfig := coreapplyv1.Service(rbg.GetWorkloadName(role), rbg.Namespace).
		WithSpec(coreapplyv1.ServiceSpec().
			WithClusterIP("None").
			WithSelector(selectMap).
			WithPublishNotReadyAddresses(true)).
		WithLabels(rbg.GetCommonLabelsFromRole(role)).
		WithAnnotations(rbg.GetCommonAnnotationsFromRole(role)).
		WithOwnerReferences(metaapplyv1.OwnerReference().
			WithAPIVersion(rbg.APIVersion).
			WithKind(rbg.Kind).
			WithName(rbg.Name).
			WithUID(rbg.GetUID()).
			WithBlockOwnerDeletion(true).
			WithController(true),
		)
	return serviceConfig, nil
}

func (r *StatefulSetReconciler) ConstructRoleStatus(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
) (workloadsv1alpha1.RoleStatus, error) {
	sts := &appsv1.StatefulSet{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, sts); err != nil {
		return workloadsv1alpha1.RoleStatus{}, err
	}
	return workloadsv1alpha1.RoleStatus{
		Name:          role.Name,
		Replicas:      *sts.Spec.Replicas,
		ReadyReplicas: sts.Status.ReadyReplicas,
	}, nil
}

func (r *StatefulSetReconciler) GetWorkloadType() string {
	return "apps/v1/StatefulSet"
}

func (r *StatefulSetReconciler) CheckWorkloadReady(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (bool, error) {
	sts := &appsv1.StatefulSet{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, sts); err != nil {
		return false, err
	}
	return sts.Status.ReadyReplicas == *sts.Spec.Replicas, nil
}
