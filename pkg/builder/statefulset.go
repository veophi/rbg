package builder

import (
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
)

type StatefulSetBuilder struct {
	Scheme *runtime.Scheme
	log    logr.Logger
}

func (b *StatefulSetBuilder) BuildStatefulSet(
	cr *workloadsv1.RoleBasedGroup,
	role workloadsv1.RoleSpec,
	injector discovery.ConfigInjector,
) (*appsv1.StatefulSet, error) {
}
