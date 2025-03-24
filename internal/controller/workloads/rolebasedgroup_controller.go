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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	workloadsv1 "sigs.k8s.io/rbgs/api/workloads/v1"
	"sigs.k8s.io/rbgs/pkg/utils"
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
	logger := log.FromContext(ctx).WithValues("leaderworkerset", klog.KObj(rbg))
	logger.Info("Starting reconciliation")

	// Initialize status if needed
	if rbg.Status.RoleStatuses == nil {
		rbg.Status.RoleStatuses = make([]workloadsv1.RoleStatus, 0)
	}

	// Process roles in dependency order
	sortedRoles, err := utils.SortRolesByDependencies(rbg)
	if err != nil {
		r.Recorder.Event(rbg, corev1.EventTypeWarning, "InvalidDependency", err.Error())
		return ctrl.Result{}, err
	}

	// Reconcile each role
	for _, role := range sortedRoles {
		// Check dependencies first
		if ready, err := utils.CheckDependencies(rbg, role); !ready || err != nil {
			if err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Dependencies not met, requeuing", "role", role.Name)
			return ctrl.Result{RequeueAfter: 5}, nil
		}

		// Reconcile StatefulSet
		if err := utils.ReconcileStatefulSet(ctx, r.Client, rbg, role, r.Scheme); err != nil {
			r.Recorder.Eventf(rbg, corev1.EventTypeWarning, "ReconcileFailed",
				"Failed to reconcile StatefulSet for role %s: %v", role.Name, err)
			return ctrl.Result{}, err
		}

		// Reconcile Service
		err := utils.CreateHeadlessServiceIfNotExists(ctx, r.Client, r.Scheme, rbg, role, logger)
		if err != nil {
			r.Recorder.Eventf(rbg, corev1.EventTypeWarning, "ReconcileFailed",
				"Failed to create Service for role %s: %v", role.Name, err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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
