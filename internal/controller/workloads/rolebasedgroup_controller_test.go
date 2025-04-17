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
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"sigs.k8s.io/rbgs/pkg/utils"
)

var _ = Describe("RoleBasedGroup Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		rolebasedgroup := &workloadsv1alpha1.RoleBasedGroup{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind RoleBasedGroup")
			err := k8sClient.Get(ctx, typeNamespacedName, rolebasedgroup)
			if err != nil && apierrors.IsNotFound(err) {
				resource := &workloadsv1alpha1.RoleBasedGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &workloadsv1alpha1.RoleBasedGroup{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance RoleBasedGroup")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			mgr, err := manager.New(cfg, manager.Options{})
			controllerReconciler := NewRoleBasedGroupReconciler(mgr)
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

func TestRoleBasedGroupReconciler_CheckCrdExists(t *testing.T) {
	// Initialize test scheme with required types
	testScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = apiextensionsv1.AddToScheme(testScheme)

	// Target CRD name according to Kubernetes naming convention
	targetCRDName := "rolebasedgroups.workloads.x-k8s.io"

	type fields struct {
		client    client.Client
		apiReader client.Reader
		scheme    *runtime.Scheme
		recorder  record.EventRecorder
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "CRD exists and ready",
			fields: fields{
				apiReader: fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(&apiextensionsv1.CustomResourceDefinition{
						ObjectMeta: metav1.ObjectMeta{Name: targetCRDName},
						Status: apiextensionsv1.CustomResourceDefinitionStatus{
							Conditions: []apiextensionsv1.CustomResourceDefinitionCondition{
								{
									Type:   apiextensionsv1.Established,
									Status: apiextensionsv1.ConditionTrue,
								},
							},
						},
					}).
					Build(),
			},
			wantErr: false,
		},
		{
			name: "CRD not found",
			fields: fields{
				apiReader: fake.NewClientBuilder().
					WithScheme(testScheme).
					Build(),
			},
			wantErr: true,
		},
		{
			name: "CRD exists but not established",
			fields: fields{
				apiReader: fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(&apiextensionsv1.CustomResourceDefinition{
						ObjectMeta: metav1.ObjectMeta{Name: targetCRDName},
						Status: apiextensionsv1.CustomResourceDefinitionStatus{
							Conditions: []apiextensionsv1.CustomResourceDefinitionCondition{
								{
									Type:   apiextensionsv1.Established,
									Status: apiextensionsv1.ConditionFalse,
								},
							},
						},
					}).
					Build(),
			},
			wantErr: true,
		},
		{
			name: "API server error",
			fields: fields{
				apiReader: utils.NewErrorInjectingClient(
					errors.New("connection refused"),
				),
			},
			wantErr: true,
		},
		{
			name: "RBAC forbidden",
			fields: fields{
				apiReader: utils.NewErrorInjectingClient(
					apierrors.NewForbidden(
						schema.GroupResource{
							Group:    "apiextensions.k8s.io",
							Resource: "customresourcedefinitions",
						},
						targetCRDName,
						errors.New("permission denied"),
					),
				),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RoleBasedGroupReconciler{
				apiReader: tt.fields.apiReader,
			}

			err := r.CheckCrdExists()

			// Verify error existence
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCrdExists() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify error types
			if tt.wantErr {
				switch tt.name {
				case "CRD not found":
					if !apierrors.IsNotFound(err) {
						t.Errorf("Expected NotFound error, got %T", err)
					}
				case "RBAC forbidden":
					if !apierrors.IsForbidden(err) {
						t.Errorf("Expected Forbidden error, got %T", err)
					}
				case "CRD exists but not established":
					if err == nil || !strings.Contains(err.Error(), "not established") {
						t.Errorf("Expected establishment error, got %v", err)
					}
				case "API server error":
					if err == nil || !strings.Contains(err.Error(), "connection refused") {
						t.Errorf("Expected connection error, got %v", err)
					}
				}
			}
		})
	}
}
