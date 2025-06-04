package reconciler

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	if err := r.reconcileStatefulSet(ctx, rbg, role); err != nil {
		return err
	}

	return r.reconcileHeadlessService(ctx, rbg, role)
}

func (r *StatefulSetReconciler) reconcileStatefulSet(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) error {
	logger := log.FromContext(ctx)
	logger.V(1).Info("start to reconciling sts workload")

	stsApplyConfig, err := r.constructStatefulSetApplyConfiguration(ctx, rbg, role)
	if err != nil {
		logger.Error(err, "Failed to construct statefulset apply configuration")
		return err
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(stsApplyConfig)
	if err != nil {
		logger.Error(err, "Converting obj apply configuration to json.")
		return err
	}

	newSts := &appsv1.StatefulSet{}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj, newSts); err != nil {
		return fmt.Errorf("convert stsApplyConfig to sts error: %s", err.Error())
	}

	oldSts := &appsv1.StatefulSet{}
	err = r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, oldSts)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	equal, err := SemanticallyEqualStatefulSet(oldSts, newSts)
	if equal {
		logger.V(1).Info("sts equal, skip reconcile")
		return nil
	}

	logger.V(1).Info(fmt.Sprintf("sts not equal, diff: %s", err.Error()))

	if err := utils.PatchObjectApplyConfiguration(ctx, r.client, stsApplyConfig, utils.PatchSpec); err != nil {
		logger.Error(err, "Failed to patch statefulset apply configuration")
		return err
	}

	return nil
}

func (r *StatefulSetReconciler) reconcileHeadlessService(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) error {
	logger := log.FromContext(ctx)
	logger.V(1).Info("start to reconciling headless service")

	sts := &appsv1.StatefulSet{}
	err := r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, sts)
	if err != nil {
		return fmt.Errorf("get sts error, skip reconcile svc. error:  %s", err.Error())
	}

	svcApplyConfig, err := r.constructServiceApplyConfiguration(ctx, rbg, role, sts)
	if err != nil {
		return err
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(svcApplyConfig)
	if err != nil {
		logger.Error(err, "Converting obj apply configuration to json.")
		return err
	}

	newSvc := &corev1.Service{}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj, newSvc); err != nil {
		return fmt.Errorf("convert svcApplyConfig to svc error: %s", err.Error())
	}

	oldSvc := &corev1.Service{}
	err = r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, oldSvc)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	equal, err := SemanticallyEqualService(oldSvc, newSvc)
	if equal {
		logger.V(1).Info("svc equal, skip reconcile")
		return nil
	}

	logger.V(1).Info(fmt.Sprintf("svc not equal, diff: %s", err.Error()))

	if err := utils.PatchObjectApplyConfiguration(ctx, r.client, svcApplyConfig, utils.PatchSpec); err != nil {
		logger.Error(err, "Failed to patch svc apply configuration")
		return err
	}

	return nil
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
			WithPodManagementPolicy(appsv1.ParallelPodManagement).
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
	sts *appsv1.StatefulSet,
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
			WithAPIVersion(sts.APIVersion).
			WithKind(sts.Kind).
			WithName(sts.Name).
			WithUID(sts.GetUID()).
			WithBlockOwnerDeletion(true),
		)
	return serviceConfig, nil
}

func (r *StatefulSetReconciler) ConstructRoleStatus(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
) (workloadsv1alpha1.RoleStatus, bool, error) {
	updateStatus := false
	sts := &appsv1.StatefulSet{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: rbg.GetWorkloadName(role), Namespace: rbg.Namespace}, sts); err != nil {
		return workloadsv1alpha1.RoleStatus{}, updateStatus, err
	}

	currentReplicas := *sts.Spec.Replicas
	currentReady := sts.Status.ReadyReplicas
	status, found := rbg.GetRoleStatus(role.Name)
	if !found || status.Replicas != currentReplicas || status.ReadyReplicas != currentReady {
		status = workloadsv1alpha1.RoleStatus{
			Name:          role.Name,
			Replicas:      currentReplicas,
			ReadyReplicas: currentReady,
		}
		updateStatus = true
	}
	return status, updateStatus, nil
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

func (r *StatefulSetReconciler) CleanupOrphanedWorkloads(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup) error {
	logger := log.FromContext(ctx)
	// list sts managed by rbg
	stsList := &appsv1.StatefulSetList{}
	if err := r.client.List(context.Background(), stsList, client.InNamespace(rbg.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": workloadsv1alpha1.ControllerName,
			"app.kubernetes.io/name":       rbg.Name,
		}),
	); err != nil {
		return err
	}

	for _, sts := range stsList.Items {
		found := false
		for _, role := range rbg.Spec.Roles {
			if role.Workload.Kind == "StatefulSet" && rbg.GetWorkloadName(&role) == sts.Name {
				found = true
				break
			}
		}
		if !found {
			if err := r.client.Delete(ctx, &sts); err != nil {
				return fmt.Errorf("delete sts %s error: %s", sts.Name, err.Error())
			}
			// The deletion of headless services depends on its own reference
			logger.Info("delete sts", "sts", sts.Name)
		}
	}
	return nil
}

func SemanticallyEqualStatefulSet(sts1, sts2 *appsv1.StatefulSet) (bool, error) {
	if sts1 == nil || sts2 == nil {
		if sts1 != sts2 {
			return false, fmt.Errorf("object is nil")
		} else {
			return true, nil
		}
	}

	if equal, err := objectMetaEqual(sts1.ObjectMeta, sts2.ObjectMeta); !equal {
		return false, fmt.Errorf("objectMeta not equal: %s", err.Error())
	}

	if equal, err := statefulSetSpecEqual(sts1.Spec, sts2.Spec); !equal {
		return false, fmt.Errorf("spec not equal: %s", err.Error())
	}
	return true, nil
}

func statefulSetSpecEqual(spec1, spec2 appsv1.StatefulSetSpec) (bool, error) {
	if spec1.Replicas != nil && spec2.Replicas != nil {
		if *spec1.Replicas != *spec2.Replicas {
			return false, fmt.Errorf("replicas not equal, old: %d, new: %d", *spec1.Replicas, *spec2.Replicas)
		}
	}

	if !reflect.DeepEqual(spec1.Selector, spec2.Selector) {
		return false, fmt.Errorf("selector not equal, old: %v, new: %v", spec1.Selector, spec2.Selector)
	}

	if spec1.ServiceName != spec2.ServiceName {
		return false, fmt.Errorf("serviceName not equal, old: %s, new: %s", spec1.ServiceName, spec2.ServiceName)
	}

	if equal, err := podTemplateSpecEqual(spec1.Template, spec2.Template); !equal {
		return false, fmt.Errorf("podTemplateSpec not equal, %s", err.Error())
	}

	return true, nil
}

func SemanticallyEqualService(svc1, svc2 *corev1.Service) (bool, error) {
	if svc1 == nil || svc2 == nil {
		if svc1 != svc2 {
			return false, fmt.Errorf("object is nil")
		} else {
			return true, nil
		}
	}

	if equal, err := objectMetaEqual(svc1.ObjectMeta, svc2.ObjectMeta); !equal {
		return false, fmt.Errorf("objectMeta not equal: %s", err.Error())
	}

	if !reflect.DeepEqual(svc1.Spec.Selector, svc2.Spec.Selector) {
		return false, fmt.Errorf("selector not equal, old: %v, new: %v", svc1.Spec.Selector, svc2.Spec.Selector)
	}

	return true, nil
}
