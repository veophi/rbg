package utils

import (
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

// TODO: use apply configuration in futrue
func ConstructStatefulsetApplyConfigurationByRole(rbg *workloadsv1alpha1.RoleBasedGroup, role workloadsv1alpha1.RoleSpec) (apply *appsapplyv1.StatefulSetApplyConfiguration, err error) {

	return
}
