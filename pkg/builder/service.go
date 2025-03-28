package builder

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/discovery"
)

type ServiceBuilder struct {
	scheme *runtime.Scheme
	client client.Client
	logger logr.Logger
}

func NewServiceBuilder(ctx context.Context, scheme *runtime.Scheme, client client.Client) *ServiceBuilder {
	return &ServiceBuilder{
		scheme: scheme,
		client: client,
		logger: log.FromContext(ctx).WithName("ServiceBuilder"),
	}
}

func (s *ServiceBuilder) Build(
	rbg *workloadsv1alpha1.RoleBasedGroup,
	role *workloadsv1alpha1.RoleSpec,
	injector discovery.GroupInjector) (obj client.Object, err error) {
	s.logger.V(1).Info("start to build service")

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

	err = controllerutil.SetControllerReference(rbg, service, s.scheme)
	if err != nil {
		return
	}

	return service, nil
}
