/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workloads

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
	"sigs.k8s.io/rbgs/pkg/builder"
	"sigs.k8s.io/rbgs/pkg/dependency"
	"sigs.k8s.io/rbgs/pkg/discovery"
	"sigs.k8s.io/rbgs/pkg/utils"
)

const (
	// FailedCreate Event reason used when a resource creation fails.
	// The event uses the error(s) as the reason.
	FailedCreate     = "FailedCreate"
	RolesProgressing = "RolesProgressing"
	RolesUpdating    = "RolesUpdating"
	CreatingRevision = "CreatingRevision"
)

// RoleBasedGroupReconciler reconciles a RoleBasedGroup object
type RoleBasedGroupReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroups/finalizers,verbs=update
func (r *RoleBasedGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the RoleBasedGroup instance
	rbg := &workloadsv1.RoleBasedGroup{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, rbg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger := log.FromContext(ctx).WithValues("rolebasedgroup", klog.KObj(rbg))
	logger.Info("Starting reconciliation")

	// Initialize status if needed
	if rbg.Status.RoleStatuses == nil {
		rbg.Status.RoleStatuses = make([]workloadsv1.RoleStatus, 0)
	}

	// Process roles in dependency order

	dependencyManager := dependency.NewtDepencyManager(logger)
	sortedRoles, err := dependencyManager.SortRoles(rbg)
	if err != nil {
		r.Recorder.Event(rbg, corev1.EventTypeWarning, "InvalidDependency", err.Error())
		return ctrl.Result{}, err
	}

	// Reconcile each role
	for _, role := range sortedRoles {
		// Check dependencies first
		if ready, err := dependencyManager.CheckDependencies(rbg, role); !ready || err != nil {
			if err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Dependencies not met, requeuing", "role", role.Name)
			return ctrl.Result{RequeueAfter: 5}, nil
		}

		// Reconcile workload
		if err := r.reconcileStatefulSet(ctx, rbg, role); err != nil {
			r.Recorder.Eventf(rbg, corev1.EventTypeWarning, "ReconcileFailed",
				"Failed to reconcile workload for role %s with type %v: %v", role.Name, role.Workload, err)
			return ctrl.Result{}, err
		}

		// Reconcile Service
		if err = r.reconcileService(ctx, rbg, role); err != nil {
			r.Recorder.Eventf(rbg, corev1.EventTypeWarning, "ReconcileFailed",
				"Failed to create Service for role %s: %v", role.Name, err)
			return ctrl.Result{}, err
		}
	}

	if err = r.updateStatus(ctx, rbg); err != nil {
		r.Recorder.Eventf(rbg, corev1.EventTypeWarning, "UpdateStatusFailed",
			"Failed to update status for %s: %v", rbg.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RoleBasedGroupReconciler) reconcileStatefulSet(
	ctx context.Context,
	rbg *workloadsv1.RoleBasedGroup,
	role *workloadsv1.RoleSpec,
) error {
	// 1. Create Builder and Injector
	builder := &builder.StatefulSetBuilder{Scheme: r.Scheme}
	injector := discovery.NewDefaultInjector(r.Client, ctx, r.Scheme)

	// 2. Build StatefulSet
	sts, err := builder.Build(ctx, rbg, role, injector)
	if err != nil {
		return err
	}

	// 3. Apply StatefulSet
	return utils.CreateOrUpdate(ctx, r.Client, sts)
}

func (r *RoleBasedGroupReconciler) reconcileService(
	ctx context.Context,
	rbg *workloadsv1.RoleBasedGroup,
	role *workloadsv1.RoleSpec,
) error {
	// 1. Create Builder and Injector
	builder := &builder.ServiceBuilder{Scheme: r.Scheme}
	injector := discovery.NewDefaultInjector(r.Client, ctx, r.Scheme)

	// 2. Build StatefulSet
	svc, err := builder.Build(ctx, rbg, role, injector)
	if err != nil {
		return err
	}

	// 3. Apply StatefulSet
	return utils.CreateOrUpdate(ctx, r.Client, svc)
}

func (r *RoleBasedGroupReconciler) updateStatus(
	ctx context.Context,
	rbg *workloadsv1.RoleBasedGroup,
) error {
	// updateStatus := false
	log := ctrl.LoggerFrom(ctx)
	oldRbg := &workloadsv1.RoleBasedGroup{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      rbg.Name,
		Namespace: rbg.Namespace,
	}, oldRbg); err != nil {
		return fmt.Errorf("failed to get fresh RBG: %w", err)
	}
	oldStatus := oldRbg.Status.DeepCopy()
	newRbg := oldRbg.DeepCopy()

	// 遍历所有角色, 构建索引
	updateStatus := false

	for _, role := range rbg.Spec.Roles {
		// 生成 StatefulSet 名称
		stsName := fmt.Sprintf("%s-%s", rbg.Name, role.Name)
		// 获取关联的 StatefulSet
		sts := &appsv1.StatefulSet{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      stsName,
			Namespace: rbg.Namespace,
		}, sts)

		if err != nil {
			return err
		}
		updateStatus = utils.UpdateRoleReplicas(newRbg, role.Name, sts)
	}

	if updateStatus {
		if reflect.DeepEqual(oldStatus, newRbg.Status) {
			log.Info("No need to update for old status %v and new status %v, because it's deepequal", oldStatus, newRbg.Status)
			return nil
		}

		if err := r.Status().Update(ctx, newRbg); err != nil {
			if !apierrors.IsConflict(err) {
				log.Error(err, "Updating LeaderWorkerSet status and/or condition.")
			}
			return err
		}
	} else {
		log.Info("No need to update for old status %v and new status %v, because updateStatus is false", oldStatus, newRbg.Status)
	}

	return nil
}

func makeCondition(conditionType workloadsv1.RoleBasedGroupConditionType) metav1.Condition {
	var condtype, reason, message string
	switch conditionType {
	case workloadsv1.RoleBasedGroupAvailable:
		condtype = string(workloadsv1.RoleBasedGroupAvailable)
		reason = "AllRolesReady"
		message = "All replicas are ready"
	case workloadsv1.RoleBasedGroupUpdateInProgress:
		condtype = string(workloadsv1.RoleBasedGroupUpdateInProgress)
		reason = RolesUpdating
		message = "Rolling Upgrade is in progress"
	default:
		condtype = string(workloadsv1.RoleBasedGroupProgressing)
		reason = RolesProgressing
		message = "Replicas are progressing"
	}

	condition := metav1.Condition{
		Type:               condtype,
		Status:             metav1.ConditionStatus(corev1.ConditionTrue),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	return condition
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleBasedGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadsv1.RoleBasedGroup{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Named("workloads-rolebasedgroup").
		Complete(r)
}
