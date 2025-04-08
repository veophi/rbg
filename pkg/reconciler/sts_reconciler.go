package reconciler

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/discovery"
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

	sts, err := r.buildSts(ctx, rbg, role)
	if err != nil {
		return err
	}
	if err := utils.CreateOrUpdate(ctx, r.client, sts); err != nil {
		return err
	}

	svc, err := r.buildHeadlessSvc(ctx, rbg, role)
	if err != nil {
		return err
	}

	return utils.CreateOrUpdate(ctx, r.client, svc)
}

func (r *StatefulSetReconciler) buildSts(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (obj client.Object, err error) {
	logger := log.FromContext(ctx)
	logger.Info("start to build sts")

	// 1. 尝试获取现有 StatefulSet
	sts := &appsv1.StatefulSet{}
	err = r.client.Get(ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-%s", rbg.Name, role.Name),
		Namespace: rbg.Namespace,
	}, sts)

	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get existing StatefulSet: %w", err)
	}

	// 2. 创建基础 StatefulSet
	if apierrors.IsNotFound(err) {
		sts = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s", rbg.Name, role.Name),
				Namespace:   rbg.Namespace,
				Labels:      rbg.GetCommonLabelsFromRole(role),
				Annotations: rbg.GetCommonAnnotationsFromRole(role),
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: fmt.Sprintf("%s-%s", rbg.Name, role.Name),
				Replicas:    role.Replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: rbg.GetCommonLabelsFromRole(role),
				},
			},
		}

		template := role.Template.DeepCopy()
		if template.ObjectMeta.Labels == nil {
			template.ObjectMeta.Labels = make(map[string]string)
		}
		if template.ObjectMeta.Annotations == nil {
			template.ObjectMeta.Annotations = make(map[string]string)
		}
		sts.Spec.Template = *template
	}

	utils.MergeMap(sts.Spec.Template.ObjectMeta.Labels, rbg.GetCommonLabelsFromRole(role))
	utils.MergeMap(sts.Spec.Template.ObjectMeta.Annotations, rbg.GetCommonAnnotationsFromRole(role))

	// 3. 注入配置
	injector := discovery.NewDefaultInjector(r.scheme, r.client)
	dummyPod := &corev1.Pod{
		ObjectMeta: *sts.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *sts.Spec.Template.Spec.DeepCopy(),
	}

	if err := injector.InjectConfig(ctx, dummyPod, rbg, role.Name, -1); err != nil {
		return nil, fmt.Errorf("failed to inject config: %w", err)
	}

	// 3.1 注入环境变量
	if err := injector.InjectEnvVars(ctx, dummyPod, rbg, role.Name, -1); err != nil {
		return nil, fmt.Errorf("failed to inject env vars: %w", err)
	}

	// 3.2. 注入sidecar
	if err := injector.InjectSidecar(ctx, dummyPod, rbg, role.Name, -1); err != nil {
		return nil, fmt.Errorf("failed to inject sidecar: %w", err)
	}

	// 4. 回写修改后的模板
	sts.Spec.Template.ObjectMeta = dummyPod.ObjectMeta
	sts.Spec.Template.Spec = dummyPod.Spec

	// 5. 设置 OwnerReference
	if err := controllerutil.SetControllerReference(rbg, sts, r.scheme); err != nil {
		return nil, err
	}

	logger.V(1).Info("Build Statefulset", "statefulset", utils.PrettyJson(sts))

	return sts, nil
}

func (r *StatefulSetReconciler) buildHeadlessSvc(
	ctx context.Context,
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec) (obj client.Object, err error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("start to build sts headless service")

	// Generate Service name (same as StatefulSet)
	svcName := fmt.Sprintf("%s-%s", rbg.Name, role.Name)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        svcName,
			Namespace:   rbg.Namespace,
			Labels:      rbg.GetCommonLabelsFromRole(role),
			Annotations: rbg.GetCommonAnnotationsFromRole(role),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None", // defines service as headless
			Selector: map[string]string{workloadsv1alpha1.SetNameLabelKey: rbg.Name,
				workloadsv1alpha1.SetRoleLabelKey: role.Name},
			PublishNotReadyAddresses: true,
		},
	}

	err = controllerutil.SetControllerReference(rbg, service, r.scheme)
	if err != nil {
		return
	}

	return service, nil
}

func (r *StatefulSetReconciler) UpdateStatus(ctx context.Context, client client.Client, rbg *workloadsv1alpha1.RoleBasedGroup, newRbg *workloadsv1alpha1.RoleBasedGroup, roleName string) (bool, error) {
	name := fmt.Sprintf("%s-%s", rbg.Name, roleName)
	sts := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: rbg.Namespace}, sts); err != nil {
		return false, err
	}

	return utils.UpdateRoleReplicas(newRbg, roleName, sts.Spec.Replicas, sts.Status.ReadyReplicas), nil
}

func (r *StatefulSetReconciler) GetWorkloadType() string {
	return "apps/v1/StatefulSet"
}
