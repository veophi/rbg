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
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/utils"
)

const (
	RoleBasedGroupSetKey = "workload-rbgs-name"
)

// RoleBasedGroupSetReconciler reconciles a RoleBasedGroupSet object
type RoleBasedGroupSetReconciler struct {
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	recorder  record.EventRecorder
}

func NewRoleBasedGroupSetReconciler(mgr ctrl.Manager) *RoleBasedGroupSetReconciler {
	return &RoleBasedGroupSetReconciler{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor("rbgset-controller"),
	}
}

// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroupsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroupsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workloads.x-k8s.io,resources=rolebasedgroupsets/finalizers,verbs=update

func (r *RoleBasedGroupSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the RoleBasedGroup instance
	rbgset := &workloadsv1alpha1.RoleBasedGroupSet{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, rbgset); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger := log.FromContext(ctx).WithValues("rbgset", klog.KObj(rbgset))
	ctx = ctrl.LoggerInto(ctx, logger)
	logger.Info("Start reconciling")

	var rbglist workloadsv1alpha1.RoleBasedGroupList
	opts := []client.ListOption{
		client.InNamespace(rbgset.Namespace),
		client.MatchingLabels{RoleBasedGroupSetKey: rbgset.Name},
	}
	err := r.client.List(ctx, &rbglist, opts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(rbglist.Items) >= int(*rbgset.Spec.Replicas) {
		// TODO update
		return ctrl.Result{}, nil
	}

	createNum := int(*rbgset.Spec.Replicas) - len(rbglist.Items)
	// create rbg
	var wg sync.WaitGroup

	for i := 1; i <= createNum; i++ {
		wg.Add(1)
		go func() {
			err := r.createRBG(ctx, rbgset, &wg)
			if err != nil {
				logger.Error(err, "create rbg failed.")
			}
		}()
	}
	wg.Wait()

	return ctrl.Result{}, nil
}

func (r *RoleBasedGroupSetReconciler) createRBG(
	ctx context.Context,
	rbgset *workloadsv1alpha1.RoleBasedGroupSet,
	wg *sync.WaitGroup) error {
	defer wg.Done()
	rbg := workloadsv1alpha1.RoleBasedGroup{}
	rbg.Namespace = rbgset.Namespace
	rbg.GenerateName = fmt.Sprintf("%s-", rbgset.Name)
	rbg.Labels = map[string]string{
		RoleBasedGroupSetKey: rbgset.Name,
	}

	if err := controllerutil.SetControllerReference(rbgset, &rbg, r.scheme); err != nil {
		return err
	}

	rbg.Spec.Roles = rbgset.Spec.Template.Roles

	err := r.client.Create(ctx, &rbg)
	if err != nil {
		return fmt.Errorf("create rbg error: %v", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleBasedGroupSetReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&workloadsv1alpha1.RoleBasedGroupSet{}).
		Owns(&workloadsv1alpha1.RoleBasedGroup{}).
		Named("workloads-rolebasedgroupset").
		Complete(r)
}

// CheckCrdExists checks if the specified Custom Resource Definition (CRD) exists in the Kubernetes cluster.
func (r *RoleBasedGroupSetReconciler) CheckCrdExists() error {
	return utils.CheckCrdExists(r.apiReader, "rolebasedgroupsets.workloads.x-k8s.io")
}
