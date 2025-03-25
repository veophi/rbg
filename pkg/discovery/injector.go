package discovery

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
	"sigs.k8s.io/rbgs/pkg/utils"
)

type ConfigInjector interface {
	InjectConfig(pod *corev1.Pod, rbg *workloadsv1.RoleBasedGroup, roleName string, index int32) error
	InjectEnvVars(pod *corev1.Pod, rbg *workloadsv1.RoleBasedGroup, roleName string, index int32) error
}

type DefaultInjector struct {
	Client client.Client
	ctx    context.Context
}

func (i *DefaultInjector) InjectConfig(pod *corev1.Pod, rbg *workloadsv1.RoleBasedGroup, roleName string, index int32) error {
	builder := &ConfigBuilder{
		// Pod:       pod,
		RBG:       rbg,
		RoleName:  roleName,
		RoleIndex: index,
	}

	configData, err := builder.Build()
	if err != nil {
		return err
	}

	cmName := fmt.Sprintf("rbgs-%s-%s-cm", rbg.GetName(), roleName)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: rbg.Namespace,
		},
		Data: map[string]string{
			"config.yaml": string(configData),
		},
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "cluster-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
				Items: []corev1.KeyToPath{
					{Key: "config.yaml", Path: "config.yaml"},
				},
			},
		},
	})

	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(
			pod.Spec.Containers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      "rbgs-group-config",
				MountPath: "/etc/rbgs",
				ReadOnly:  true,
			},
		)
	}

	// Create/Update ConfigMap logic
	// return reconcileConfigMap(i.Client, rbg, configData)
	return utils.CreateOrUpdate(i.ctx, i.Client, configMap)
}

func (i *DefaultInjector) InjectEnvVars(pod *corev1.Pod, rbg client.Object, roleName string, index int32) error {
	generator := &EnvGenerator{
		RBG:       rbg.(*workloadsv1.RoleBasedGroup),
		RoleName:  roleName,
		RoleIndex: index,
	}

	envVars := generator.Generate()
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Env = append(
			pod.Spec.Containers[i].Env,
			envVars...,
		)
	}
	return nil
}
