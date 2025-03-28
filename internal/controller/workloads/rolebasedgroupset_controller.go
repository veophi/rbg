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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sync"
)

const (
	RoleBasedGroupSetKey = "workload-rbgs-name"
)

// RoleBasedGroupSetReconciler reconciles a RoleBasedGroupSet object
type RoleBasedGroupSetReconciler struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	logger   logr.Logger
}

func NewRoleBasedGroupSetReconciler(mgr ctrl.Manager) *RoleBasedGroupSetReconciler {
	return &RoleBasedGroupSetReconciler{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor("rbgs-controller"),
		logger:   mgr.GetLogger().WithName("rbgs-controller"),
	}
}

// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroupsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroupsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroupsets/finalizers,verbs=update

func (r *RoleBasedGroupSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the RoleBasedGroup instance
	rbgs := &workloadsv1alpha1.RoleBasedGroupSet{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, rbgs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.logger = r.logger.WithValues("rbgs", klog.KObj(rbgs))
	r.logger.Info("Starting reconciliation")

	var rbglist workloadsv1alpha1.RoleBasedGroupList
	opts := []client.ListOption{
		client.InNamespace(rbgs.Namespace),
		client.MatchingLabels{RoleBasedGroupSetKey: rbgs.Name},
	}
	err := r.client.List(context.TODO(), &rbglist, opts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(rbglist.Items) >= int(*rbgs.Spec.Replicas) {
		// TODO update
		return ctrl.Result{}, nil
	}

	createNum := int(*rbgs.Spec.Replicas) - len(rbglist.Items)
	// create rbg
	var wg sync.WaitGroup

	for i := 1; i <= createNum; i++ {
		wg.Add(1)
		go func() {
			err := createRBG(r.client, rbgs, r.scheme, &wg)
			if err != nil {
				r.logger.Error(err, "create rbg failed.")
			}
		}()
	}
	wg.Wait()

	return ctrl.Result{}, nil
}

func createRBG(client client.Client, rbgs *workloadsv1alpha1.RoleBasedGroupSet, schema *runtime.Scheme, wg *sync.WaitGroup) error {
	defer wg.Done()
	rbg := workloadsv1alpha1.RoleBasedGroup{}
	rbg.Namespace = rbgs.Namespace
	rbg.GenerateName = fmt.Sprintf("%s-", rbgs.Name)
	rbg.Labels = map[string]string{
		RoleBasedGroupSetKey: rbgs.Name,
	}

	if err := controllerutil.SetControllerReference(rbgs, &rbg, schema); err != nil {
		return err
	}

	rbg.Spec.Roles = rbgs.Spec.Template.Roles

	err := client.Create(context.TODO(), &rbg)
	if err != nil {
		return fmt.Errorf("create rbg error: %v", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleBasedGroupSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
			r.logger.Info(e.Object.GetName(), "type", reflect.ValueOf(e.Object))
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadsv1alpha1.RoleBasedGroupSet{}).
		Owns(&workloadsv1alpha1.RoleBasedGroup{}).
		Named("workloads-rbgs").
		WithEventFilter(pred).
		Complete(r)
}
