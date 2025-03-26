package discovery

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
	"sigs.k8s.io/rbgs/pkg/utils"
)

type ConfigInjector interface {
	InjectConfig(pod *corev1.Pod, rbg *workloadsv1.RoleBasedGroup, roleName string, index int32) error
	InjectEnvVars(pod *corev1.Pod, rbg *workloadsv1.RoleBasedGroup, roleName string, index int32) error
}

type DefaultInjector struct {
	client client.Client
	ctx    context.Context
	scheme *runtime.Scheme
}

func NewDefaultInjector(client client.Client, ctx context.Context, scheme *runtime.Scheme) *DefaultInjector {
	return &DefaultInjector{
		client: client,
		ctx:    ctx,
		scheme: scheme,
	}
}

func (i *DefaultInjector) InjectConfig(pod *corev1.Pod, rbg *workloadsv1.RoleBasedGroup, roleName string, index int32) error {
	builder := &ConfigBuilder{
		// Pod:       pod,
		RBG:       rbg,
		GroupName: rbg.GetName(),
		RoleName:  roleName,
		RoleIndex: index,
	}
	const (
		volumeName = "rbg-cluster-config"
		mountPath  = "/etc/rbg"
		configKey  = "config.yaml"
	)

	configData, err := builder.Build()
	if err != nil {
		return err
	}

	cmName := fmt.Sprintf("%s-%s-rbg-cm", rbg.GetName(), roleName)

	configMap := &corev1.ConfigMap{}
	err = i.client.Get(i.ctx, types.NamespacedName{
		Namespace: rbg.Namespace,
		Name:      cmName,
	}, configMap)
	if err != nil && apierrors.IsNotFound(err) {
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: rbg.Namespace,
			}, Data: make(map[string]string),
		}

		// 设置 OwnerReference
		if err := controllerutil.SetControllerReference(rbg, configMap, i.scheme); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	configMap.Data[configKey] = string(configData)

	volumeExists := false
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == volumeName {
			volumeExists = true
			break
		}
	}
	if !volumeExists {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cmName,
					},
					Items: []corev1.KeyToPath{
						{Key: configKey, Path: configKey},
					},
				},
			},
		})
	}

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		mountExists := false
		for _, vm := range container.VolumeMounts {
			if vm.Name == volumeName && vm.MountPath == mountPath {
				mountExists = true
				break
			}
		}
		if !mountExists {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      volumeName,
				MountPath: mountPath,
				ReadOnly:  true,
			})
		}
	}

	// Create/Update ConfigMap logic
	// return reconcileConfigMap(i.Client, rbg, configData)
	return utils.CreateOrUpdate(i.ctx, i.client, configMap)
}

func (i *DefaultInjector) InjectEnvVars(pod *corev1.Pod, rbg *workloadsv1.RoleBasedGroup, roleName string, index int32) error {
	generator := &EnvGenerator{
		RBG:       rbg,
		RoleName:  roleName,
		RoleIndex: index,
	}

	envVars := generator.Generate()

	for idx := range pod.Spec.Containers {
		container := &pod.Spec.Containers[idx]
		// 1. 将现有环境变量转为 Map 去重
		existingEnv := make(map[string]corev1.EnvVar)
		for _, e := range container.Env {
			existingEnv[e.Name] = e
		}
		// 2. 合并新环境变量（同名覆盖）
		for _, newEnv := range envVars {
			existingEnv[newEnv.Name] = newEnv // 新变量覆盖旧值
		}
		// 3. 将 Map 转换回 Slice
		mergedEnv := make([]corev1.EnvVar, 0, len(existingEnv))
		for _, env := range existingEnv {
			mergedEnv = append(mergedEnv, env)
		}
		container.Env = mergedEnv
	}
	return nil
}
