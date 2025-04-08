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
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/discovery"
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
	deploy, err := r.buildDeploy(ctx, rbg, role)
	if err != nil {
		return err
	}

	return utils.CreateOrUpdate(ctx, r.client, deploy)
}

func (r *DeploymentReconciler) buildDeploy(ctx context.Context, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec) (client.Object, error) {
	logger := log.FromContext(ctx)

	// 1. 尝试获取现有 Deployment
	deployment := &appsv1.Deployment{}
	err := r.client.Get(ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-%s", rbg.Name, role.Name),
		Namespace: rbg.Namespace,
	}, deployment)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get existing Deployment: %w", err)
	}

	// 2. 创建基础 Deployment
	if apierrors.IsNotFound(err) {
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s", rbg.Name, role.Name),
				Namespace:   rbg.Namespace,
				Labels:      rbg.GetCommonLabelsFromRole(role),
				Annotations: rbg.GetCommonAnnotationsFromRole(role),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: role.Replicas,
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
		deployment.Spec.Template = *template
	}

	utils.MergeMap(deployment.Spec.Template.ObjectMeta.Labels, rbg.GetCommonLabelsFromRole(role))
	utils.MergeMap(deployment.Spec.Template.ObjectMeta.Annotations, rbg.GetCommonAnnotationsFromRole(role))

	// 3. 注入配置
	injector := discovery.NewDefaultInjector(r.scheme, r.client)
	dummyPod := &corev1.Pod{
		ObjectMeta: *deployment.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *deployment.Spec.Template.Spec.DeepCopy(),
	}

	if err := injector.InjectConfig(ctx, dummyPod, rbg, role.Name, -1); err != nil {
		return nil, fmt.Errorf("failed to inject config: %w", err)
	}

	// 3.1 注入环境变量
	if err := injector.InjectEnvVars(ctx, dummyPod, rbg, role.Name, -1); err != nil {
		return nil, fmt.Errorf("failed to inject env vars: %w", err)
	}

	// 3.2 注入sidecar
	if err := injector.InjectSidecar(ctx, dummyPod, rbg, role.Name, -1); err != nil {
		return nil, fmt.Errorf("failed to inject sidecar: %w", err)
	}

	// 4. 回写修改后的模板
	deployment.Spec.Template.ObjectMeta = dummyPod.ObjectMeta
	deployment.Spec.Template.Spec = dummyPod.Spec

	// 5. 设置 OwnerReference
	if err := controllerutil.SetControllerReference(rbg, deployment, r.scheme); err != nil {
		return nil, err
	}

	logger.Info("Build Deployment", "deployment", klog.KObj(deployment))
	return deployment, nil
}

func (r *DeploymentReconciler) UpdateStatus(ctx context.Context, client client.Client, rbg *workloadsv1alpha1.RoleBasedGroup, newRbg *workloadsv1alpha1.RoleBasedGroup, roleName string) (bool, error) {
	name := fmt.Sprintf("%s-%s", rbg.Name, roleName)
	deploy := &appsv1.Deployment{}
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: rbg.Namespace}, deploy); err != nil {
		return false, err
	}
	return utils.UpdateRoleReplicas(newRbg, roleName, deploy.Spec.Replicas, deploy.Status.ReadyReplicas), nil
}

func (r *DeploymentReconciler) GetWorkloadType() string {
	return "apps/v1/Deployment"
}
