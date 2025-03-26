package builder

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
	"sigs.k8s.io/rbgs/pkg/discovery"
	"sigs.k8s.io/rbgs/pkg/utils"
)

type StatefulSetBuilder struct {
	Scheme *runtime.Scheme
	log    logr.Logger
}

func (b *StatefulSetBuilder) Build(ctx context.Context,
	rbg *workloadsv1.RoleBasedGroup,
	role *workloadsv1.RoleSpec,
	injector discovery.ConfigInjector) (obj client.Object, err error) {
	b.log.Info("start loging")

	// 1. 创建基础 StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", rbg.Name, role.Name),
			Namespace:   rbg.Namespace,
			Labels:      rbg.GetCommonLabelsFromRole(role),
			Annotations: rbg.GetCommonAnnotationsFromRole(role),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: role.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: rbg.GetCommonLabelsFromRole(role),
			},
			// Template: corev1.PodTemplateSpec{
			// 	ObjectMeta: metav1.ObjectMeta{
			// 		Labels: rbg.GetCommonLabelsFromRole(role),
			// 	},
			// 	Spec: &role.Template.Spec.DeepCopy(),
			// },
			Template: *role.Template.DeepCopy(),
		},
	}

	utils.MergeMap(sts.Spec.Template.ObjectMeta.Labels, rbg.GetCommonLabelsFromRole(role))
	utils.MergeMap(sts.Spec.Template.ObjectMeta.Annotations, rbg.GetCommonAnnotationsFromRole(role))

	// 2. 注入配置
	dummyPod := &corev1.Pod{
		ObjectMeta: sts.Spec.Template.ObjectMeta,
		Spec:       sts.Spec.Template.Spec,
	}

	if err := injector.InjectConfig(dummyPod, rbg, role.Name, 0); err != nil {
		return nil, fmt.Errorf("failed to inject config: %w", err)
	}

	// 3. 注入环境变量
	if err := injector.InjectEnvVars(dummyPod, rbg, role.Name, 0); err != nil {
		return nil, fmt.Errorf("failed to inject env vars: %w", err)
	}

	// 4. 回写修改后的模板
	sts.Spec.Template.ObjectMeta = dummyPod.ObjectMeta
	sts.Spec.Template.Spec = dummyPod.Spec

	// 5. 设置 OwnerReference
	if err := controllerutil.SetControllerReference(rbg, sts, b.Scheme); err != nil {
		return nil, err
	}

	b.log.Info("Statefulset: %v", sts)

	return sts, nil
}
