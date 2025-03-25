package builder

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
)

type ResourceBuilder interface {
	build(cr *workloadsv1.RoleBasedGroup, role workloadsv1.RoleSpec, injector discovery.ConfigInjector) (obj client.Object, err error)
}
