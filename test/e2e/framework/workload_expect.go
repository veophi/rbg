package framework

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

type WorkloadEqualChecker interface {
	ExpectWorkloadEqual(rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec) error
	GetWorkload() string
}

func NewWorkloadEqualChecker(ctx context.Context, client client.Client, apiVersion, kind string) (WorkloadEqualChecker, error) {
	switch {
	case apiVersion == "apps/v1" && kind == "Deployment":
		return NewDeploymentEqualChecker(ctx, client), nil
	case apiVersion == "apps/v1" && kind == "StatefulSet":
		return NewStatefulSetEqualChecker(ctx, client), nil
	default:
		return nil, fmt.Errorf("unsupported workload type: %s/%s", apiVersion, kind)
	}
}

type DeploymentEqualChecker struct {
	ctx    context.Context
	client client.Client
}

var _ WorkloadEqualChecker = &DeploymentEqualChecker{}

func NewDeploymentEqualChecker(ctx context.Context, client client.Client) *DeploymentEqualChecker {
	return &DeploymentEqualChecker{
		ctx:    ctx,
		client: client,
	}
}

func (d *DeploymentEqualChecker) ExpectWorkloadEqual(rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec) error {
	// 1. 获取deployment
	deployment := &appsv1.Deployment{}
	err := d.client.Get(d.ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-%s", rbg.Name, role.Name),
		Namespace: rbg.Namespace,
	}, deployment)
	if err != nil {
		return fmt.Errorf("failed to get existing Deployment: %w", err)
	}
	return nil
}
func (d *DeploymentEqualChecker) GetWorkload() string {
	return "apps/v1/Deployment"
}

type StatefulSetEqualChecker struct {
	ctx    context.Context
	client client.Client
}

var _ WorkloadEqualChecker = &StatefulSetEqualChecker{}

func NewStatefulSetEqualChecker(ctx context.Context, client client.Client) *StatefulSetEqualChecker {
	return &StatefulSetEqualChecker{
		ctx:    ctx,
		client: client,
	}
}

func (s *StatefulSetEqualChecker) ExpectWorkloadEqual(rbg *v1alpha1.RoleBasedGroup, role v1alpha1.RoleSpec) error {
	// 1. 获取sts
	sts := &appsv1.StatefulSet{}
	err := s.client.Get(s.ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-%s", rbg.Name, role.Name),
		Namespace: rbg.Namespace,
	}, sts)
	if err != nil {
		return fmt.Errorf("failed to get existing StatefulSet: %w", err)
	}

	// 2. 获取service
	svc := &v1.Service{}
	err = s.client.Get(s.ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-%s", rbg.Name, role.Name),
		Namespace: rbg.Namespace,
	}, svc)
	if err != nil {
		return fmt.Errorf("failed to get existing headless svc: %w", err)
	}
	return nil
}
func (s *StatefulSetEqualChecker) GetWorkload() string {
	return "apps/v1/StatefulSet"
}
