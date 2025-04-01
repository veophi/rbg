package discovery

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	workloadsv1alpha "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

const (
	RuntimeContainerName = "patio-runtime"
	// TODO 替换为正式版本镜像
	RuntimeContainerImage       = "registry.ap-southeast-1.aliyuncs.com/zibai-test/patio-runtime:v1"
	RuntimeGroupConfigFileName  = "patio-group-config"
	RuntimeGroupConfigMountPath = "/etc/patio"
)

type SidecarBuilder struct {
	rbg       *workloadsv1alpha.RoleBasedGroup
	roleIndex int32
	roleName  string
}

func NewSidecarBuilder(rbg *workloadsv1alpha.RoleBasedGroup, roleName string) *SidecarBuilder {
	return &SidecarBuilder{
		rbg:      rbg,
		roleName: roleName,
	}
}

func (builder *SidecarBuilder) Build(pod *v1.Pod) error {

	var curRole workloadsv1alpha.RoleSpec
	found := false

	for _, role := range builder.rbg.Spec.Roles {
		if role.Name == builder.roleName {
			curRole = role
			found = true
			break
		}
	}

	if !found || curRole.RuntimeEngine == nil {
		return nil
	}

	containerImage := RuntimeContainerImage
	if curRole.RuntimeEngine.Image != "" {
		containerImage = curRole.RuntimeEngine.Image
	}

	// 1. add runtime container
	c := &v1.Container{
		Name:  RuntimeContainerName,
		Image: containerImage,
		Ports: []v1.ContainerPort{
			{
				ContainerPort: 8080,
				Protocol:      v1.ProtocolTCP,
			},
		},
		Resources: v1.ResourceRequirements{
			Limits: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU:    resource.MustParse("100m"),
				v1.ResourceMemory: resource.MustParse("100Mi"),
			},
		},
	}

	c.Args = append(c.Args, curRole.RuntimeEngine.Args...)

	c.Env = append(c.Env, curRole.RuntimeEngine.Env...)

	pod.Spec.Containers = append(pod.Spec.Containers, *c)

	// 2. add volume & volumeMount
	if curRole.RuntimeEngine.MountGroupConfig {
		// volume
		configVolume := v1.Volume{
			Name: RuntimeGroupConfigFileName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, configVolume)

		// volume mount
		mountPath := RuntimeGroupConfigMountPath
		if curRole.RuntimeEngine.GroupConfigMountPath != "" {
			mountPath = curRole.RuntimeEngine.GroupConfigMountPath
		}
		configVolumeMount := v1.VolumeMount{
			Name:      RuntimeGroupConfigFileName,
			MountPath: mountPath,
		}

		for i := range pod.Spec.Containers {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, configVolumeMount)
		}
	}

	return nil
}
