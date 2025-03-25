package builder

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
	"sigs.k8s.io/rbgs/pkg/utils"
)

type defaultBuilder struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func NewDefaultBuilder(client client.Client, scheme *runtime.Scheme, log logr.Logger) *defaultBuilder {
	return &defaultBuilder{
		client: client,
		scheme: scheme,
		log:    log,
	}
}

func (builder *defaultBuilder) ReconcileWorkloadByRole(ctx context.Context, rbg *workloadsv1.RoleBasedGroup, role *workloadsv1.RoleSpec) (err error) {
	// Generate StatefulSet name
	stsName := fmt.Sprintf("%s-%s", rbg.Name, role.Name)

	// Create or update StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        stsName,
			Namespace:   rbg.Namespace,
			Labels:      rbg.GetCommonLabelsFromRole(role),
			Annotations: rbg.GetCommonAnnotationsFromRole(role),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            role.Replicas,
			Template:            role.Template.Spec,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.OnDeleteStatefulSetStrategyType,
			},
		},
	}

	// Set controller reference
	err = ctrl.SetControllerReference(rbg, sts, builder.scheme)
	if err != nil {
		return
	}

	// Inject environment variables
	envVars := injectEnvironmentVariableForLocalRole(role)
	// Add environment variables to all containers
	for i := range sts.Spec.Template.Spec.Containers {
		sts.Spec.Template.Spec.Containers[i].Env = append(
			sts.Spec.Template.Spec.Containers[i].Env,
			envVars...,
		)
	}

	return utils.CreateOrUpdate(ctx, builder.client, sts)
}
