package discovery

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
)

type EnvGenerator struct {
	RBG       *workloadsv1.RoleBasedGroup
	RoleName  string
	RoleIndex int32
}

func (g *EnvGenerator) Generate() []corev1.EnvVar {
	// return []corev1.EnvVar{
	// 	g.buildRoleSizeVar(),
	// 	g.buildRoleAddressVars()
	// }
	envVars := make([]corev1.EnvVar, 0)
	envVars = append(envVars, g.buildRoleSizeVar())
	envVars = append(envVars, g.buildLocalRoleVars()...)
	envVars = append(envVars, g.buildRoleAddressVars()...)
	return envVars
}

func (g *EnvGenerator) buildRoleSizeVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  fmt.Sprintf("ROLES_%s_SIZE", strings.ToUpper(g.RoleName)),
		Value: fmt.Sprintf("%d", *g.RBG.GetRole(g.RoleName).Replicas),
	}
}

func (g *EnvGenerator) buildRoleAddressVars() []corev1.EnvVar {
	var envVars []corev1.EnvVar
	for _, role := range g.RBG.Spec.Roles {
		serviceName := fmt.Sprintf("%s-%s", g.RBG.Name, role.Name)
		for i := 0; i < int(*role.Replicas); i++ {
			basePrefix := fmt.Sprintf("ROLES_%s_%d", strings.ToUpper(role.Name), i)

			// 地址变量
			envVars = append(envVars, corev1.EnvVar{
				Name:  basePrefix + "_ADDRESS",
				Value: fmt.Sprintf("%s-%d.%s", role.Name, i, serviceName),
			})

			// 多端口变量
			// # 基础地址
			// ROLES_PREFILL_0_ADDRESS=prefill-0.deepseek-0-prefill

			// # 多端口变量（假设 ServicePorts 定义如下）：
			// # - name: http, port: 8080
			// # - name: metrics, port: 9090
			// ROLES_PREFILL_0_HTTP_PORT=8080
			// ROLES_PREFILL_0_METRICS_PORT=9090

			// # 匿名端口处理（port 未命名）
			// ROLES_DECODE_0_PORT50051=50051
			// ROLES_DECODE_0_PORT8080=8080
			for _, port := range role.ServicePorts {
				portKey := generatePortKey(port)
				envVars = append(envVars,
					corev1.EnvVar{
						Name:  fmt.Sprintf("%s_%s_PORT", basePrefix, portKey),
						Value: fmt.Sprintf("%d", port.Port),
					},
				)
			}
		}
	}
	return envVars
}

func (g *EnvGenerator) buildLocalRoleVars() (envVars []corev1.EnvVar) {
	// Inject environment variables for service discovery
	envVars = []corev1.EnvVar{
		{
			Name:  "ROLE_NAME",
			Value: g.RoleName,
		},
		// for statefulset
		{
			Name: "ROLE_INDEX",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.annotations['apps.kubernetes.io/pod-index']",
				},
			},
		},
	}

	return
}
