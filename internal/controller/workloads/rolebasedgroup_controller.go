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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/rbgs/pkg/utils"

	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/dependency"
	"sigs.k8s.io/rbgs/pkg/reconciler"
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
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func NewRoleBasedGroupReconciler(mgr ctrl.Manager) *RoleBasedGroupReconciler {
	return &RoleBasedGroupReconciler{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor("RoleBasedGroup"),
	}
}

// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroups/finalizers,verbs=update
func (r *RoleBasedGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the RoleBasedGroup instance
	rbg := &workloadsv1alpha1.RoleBasedGroup{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, rbg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	newRbg := rbg.DeepCopy()
	oldStatus := rbg.Status.DeepCopy()

	logger := log.FromContext(ctx).WithValues("rbg", klog.KObj(rbg))
	ctx = ctrl.LoggerInto(ctx, logger)
	logger.Info("Start reconciling")

	// Initialize status if needed
	if rbg.Status.RoleStatuses == nil {
		rbg.Status.RoleStatuses = make([]workloadsv1alpha1.RoleStatus, 0)
	}

	// Process roles in dependency order
	dependencyManager := dependency.NewDependencyManager()
	sortedRoles, err := dependencyManager.SortRoles(rbg)
	if err != nil {
		r.recorder.Event(rbg, corev1.EventTypeWarning, "InvalidDependency", err.Error())
		return ctrl.Result{}, err
	}

	// Reconcile each role
	needUpdateStatus := false
	for _, role := range sortedRoles {
		// Check dependencies first
		if ready, err := dependencyManager.CheckDependencies(rbg, role); !ready || err != nil {
			if err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Dependencies not met, requeuing", "role", role.Name)
			return ctrl.Result{RequeueAfter: 5}, nil
		}

		logger.Info("start reconcile workload")
		reconciler, err := reconciler.NewWorkloadReconciler(
			role.Workload.APIVersion,
			role.Workload.Kind,
			r.scheme,
			r.client,
		)
		if err != nil {
			logger.Error(err, "Failed to create workload reconciler")
			return ctrl.Result{}, err

		}

		if err := reconciler.Reconciler(ctx, rbg, role); err != nil {
			r.recorder.Eventf(rbg, corev1.EventTypeWarning, "ReconcileFailed",
				"Failed to reconcile %s for role %s: %v", reconciler.GetWorkloadType(), role.Name, err)
			return ctrl.Result{}, err
		}

		needUpdateStatus, err = reconciler.UpdateStatus(ctx, r.client, rbg, newRbg, role.Name)
		if err != nil {
			r.recorder.Eventf(rbg, corev1.EventTypeWarning, "UpdateStatusFailed",
				"Failed to update status for %s: %v", rbg.Name, err)
			return ctrl.Result{}, err
		}
	}

	if needUpdateStatus {
		if err := utils.UpdateRbgStatus(ctx, r.client, oldStatus, newRbg); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		logger.V(1).Info("No need to update for old status  and new status , because it's deepequal", "oldStatus", oldStatus, "newStatus", newRbg.Status)
	}

	return ctrl.Result{}, nil
}

func makeCondition(conditionType workloadsv1alpha1.RoleBasedGroupConditionType) metav1.Condition {
	var condtype, reason, message string
	switch conditionType {
	case workloadsv1alpha1.RoleBasedGroupAvailable:
		condtype = string(workloadsv1alpha1.RoleBasedGroupAvailable)
		reason = "AllRolesReady"
		message = "All replicas are ready"
	case workloadsv1alpha1.RoleBasedGroupUpdateInProgress:
		condtype = string(workloadsv1alpha1.RoleBasedGroupUpdateInProgress)
		reason = RolesUpdating
		message = "Rolling Upgrade is in progress"
	default:
		condtype = string(workloadsv1alpha1.RoleBasedGroupProgressing)
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
	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldRbg, ok := e.ObjectOld.(*workloadsv1alpha1.RoleBasedGroup)
			if !ok {
				return false
			}
			newRbg := e.ObjectNew.(*workloadsv1alpha1.RoleBasedGroup)
			return !reflect.DeepEqual(oldRbg.Spec, newRbg.Spec)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.(*workloadsv1alpha1.RoleBasedGroup)
			return ok
		},
		DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadsv1alpha1.RoleBasedGroup{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Named("workloads-rolebasedgroup").
		WithEventFilter(pred).
		Complete(r)
}
