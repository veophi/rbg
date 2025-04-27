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
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/dependency"
	"sigs.k8s.io/rbgs/pkg/reconciler"
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
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	recorder  record.EventRecorder
}

func NewRoleBasedGroupReconciler(mgr ctrl.Manager) *RoleBasedGroupReconciler {
	return &RoleBasedGroupReconciler{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor("RoleBasedGroup"),
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

	logger := log.FromContext(ctx).WithValues("rbg", klog.KObj(rbg))
	ctx = ctrl.LoggerInto(ctx, logger)
	logger.Info("Start reconciling")

	// Process roles in dependency order
	dependencyManager := dependency.NewDefaultDependencyManager(r.scheme, r.client)
	sortedRoles, err := dependencyManager.SortRoles(ctx, rbg)
	if err != nil {
		r.recorder.Event(rbg, corev1.EventTypeWarning, "InvalidDependency", err.Error())
		return ctrl.Result{}, err
	}

	// Reconcile each role
	var roleStatuses []workloadsv1alpha1.RoleStatus
	for _, role := range sortedRoles {
		// Check dependencies first
		if ready, err := dependencyManager.CheckDependencyReady(ctx, rbg, role); !ready || err != nil {
			if err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("Dependencies not met, requeuing", "role", role.Name)
			return ctrl.Result{RequeueAfter: 5}, nil
		}

		logger.Info("start reconcile workload")
		reconciler, err := reconciler.NewWorkloadReconciler(role.Workload.APIVersion, role.Workload.Kind, r.scheme, r.client)
		if err != nil {
			logger.Error(err, "Failed to create workload reconciler")
			return ctrl.Result{}, err
		}

		if err := reconciler.Reconciler(ctx, rbg, role); err != nil {
			r.recorder.Eventf(rbg, corev1.EventTypeWarning, "ReconcileFailed",
				"Failed to reconcile %s for role %s: %v", reconciler.GetWorkloadType(), role.Name, err)
			return ctrl.Result{}, err
		}

		roleStatus, err := reconciler.ConstructRoleStatus(ctx, rbg, role)
		if err != nil {
			logger.Error(err, "Failed to construct role status")
			return ctrl.Result{}, err
		}
		roleStatuses = append(roleStatuses, roleStatus)
	}

	// update rbg status
	rbgApplyConfig := utils.RoleBasedGroup(rbg.Name, rbg.Namespace, rbg.Kind, rbg.APIVersion).
		WithStatus(utils.RbgStatus().WithRoleStatuses(roleStatuses...))

	if err := utils.PatchObjectApplyConfiguration(ctx, r.client, rbgApplyConfig, utils.PatchStatus); err != nil {
		r.recorder.Eventf(rbg, corev1.EventTypeWarning, "Update role status error",
			"Failed to update status for %s: %v", rbg.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleBasedGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadsv1alpha1.RoleBasedGroup{}, builder.WithPredicates(RBGPredicate())).
		Owns(&appsv1.StatefulSet{}, builder.WithPredicates(WorkloadPredicate())).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(WorkloadPredicate())).
		Owns(&corev1.Service{}).
		Named("workloads-rolebasedgroup").
		Complete(r)
}

// CheckCrdExists checks if the specified Custom Resource Definition (CRD) exists in the Kubernetes cluster.
func (r *RoleBasedGroupReconciler) CheckCrdExists() error {
	return utils.CheckCrdExists(r.apiReader, "rolebasedgroups.workloads.x-k8s.io")
}

func RBGPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.(*workloadsv1alpha1.RoleBasedGroup)
			if ok {
				ctrl.Log.Info("enqueue: rbg create event", "rbg", klog.KObj(e.Object))
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldRbg, ok1 := e.ObjectOld.(*workloadsv1alpha1.RoleBasedGroup)
			newRbg, ok2 := e.ObjectNew.(*workloadsv1alpha1.RoleBasedGroup)
			if ok1 && ok2 {
				if !reflect.DeepEqual(oldRbg.Spec, newRbg.Spec) {
					ctrl.Log.Info("enqueue: rbg update event", "rbg", klog.KObj(e.ObjectOld))
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			_, ok := e.Object.(*workloadsv1alpha1.RoleBasedGroup)
			if ok {
				ctrl.Log.Info("enqueue: rbg delete event", "rbg", klog.KObj(e.Object))
				return true
			}
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func WorkloadPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// ignore workload create event
			return false
		},
		UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
			if e.ObjectOld.GetOwnerReferences() == nil || len(e.ObjectOld.GetOwnerReferences()) == 0 ||
				e.ObjectNew.GetOwnerReferences() == nil || len(e.ObjectNew.GetOwnerReferences()) == 0 {
				return false
			}

			rbg := workloadsv1alpha1.RoleBasedGroup{}
			if !utils.CheckOwnerReference(e.ObjectOld.GetOwnerReferences(), rbg.GroupVersionKind()) ||
				utils.CheckOwnerReference(e.ObjectNew.GetOwnerReferences(), rbg.GroupVersionKind()) {
				return false
			}

			equal, err := reconciler.WorkloadEqual(e.ObjectOld, e.ObjectNew)
			if !equal {
				if err != nil {
					ctrl.Log.Info("enqueue: workload update event",
						"rbg", klog.KObj(e.ObjectOld), "diff", err.Error())
				}
				return true
			}

			return false
		},
		DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
			if e.Object.GetOwnerReferences() == nil || len(e.Object.GetOwnerReferences()) == 0 {
				return false
			}

			rbg := workloadsv1alpha1.RoleBasedGroup{}
			if !utils.CheckOwnerReference(e.Object.GetOwnerReferences(), rbg.GroupVersionKind()) {
				return false
			}

			ctrl.Log.Info("enqueue: workload delete event", "rbg", klog.KObj(e.Object))
			return true
		},
		GenericFunc: func(e event.TypedGenericEvent[client.Object]) bool {
			return false
		},
	}
}
