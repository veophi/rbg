package utils

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

func CreateHeadlessServiceIfNotExists(ctx context.Context, k8sClient client.Client, Scheme *runtime.Scheme, rbg *workloadsv1alpha1.RoleBasedGroup, role *workloadsv1alpha1.RoleSpec, log logr.Logger) (err error) {
	// Generate Service name (same as StatefulSet)
	svcName := fmt.Sprintf("%s-%s", rbg.Name, role.Name)

	var headlessService corev1.Service
	if err = k8sClient.Get(ctx, types.NamespacedName{Name: svcName, Namespace: rbg.GetNamespace()}, &headlessService); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		headlessService := corev1.Service{
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

		if err := ctrl.SetControllerReference(rbg, &headlessService, Scheme); err != nil {
			return err
		}
		// create the service in the cluster
		log.V(2).Info("Creating headless service.")
		if err := k8sClient.Create(ctx, &headlessService); err != nil {
			return err
		}
	}

	return
}
