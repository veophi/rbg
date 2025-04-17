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
	"errors"
	"testing"

	// . "github.com/onsi/ginkgo/v2"
	// . "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/rbgs/pkg/utils"
)

func TestRoleBasedGroupSetReconciler_CheckCrdExists(t *testing.T) {
	// Setup test scheme with required types
	testScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = apiextensionsv1.AddToScheme(testScheme)

	// Target CRD name following Kubernetes naming convention
	targetCRDName := "rolebasedgroupsets.workloads.x-k8s.io"

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
			name: "CRD exists and established",
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
			name: "CRD not found in cluster",
			fields: fields{
				apiReader: fake.NewClientBuilder().
					WithScheme(testScheme).
					Build(), // Empty client
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
						Status:     apiextensionsv1.CustomResourceDefinitionStatus{
							// No established condition
						},
					}).
					Build(),
			},
			wantErr: true,
		},
		{
			name: "API server connection error",
			fields: fields{
				apiReader: utils.NewErrorInjectingClient(
					errors.New("connection refused"), // Simulate API server downtime
				),
			},
			wantErr: true,
		},
		{
			name: "RBAC permission denied",
			fields: fields{
				apiReader: utils.NewErrorInjectingClient(
					apierrors.NewForbidden(
						schema.GroupResource{Group: "apiextensions.k8s.io", Resource: "customresourcedefinitions"},
						targetCRDName,
						errors.New("requires 'get' permission"),
					),
				),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RoleBasedGroupSetReconciler{
				apiReader: tt.fields.apiReader, // Focus on API reader component
			}

			// Execute the check operation
			err := r.CheckCrdExists()

			// Validate error expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCrdExists() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Additional error type assertions
			if tt.wantErr {
				switch tt.name {
				case "RBAC permission denied":
					if !apierrors.IsForbidden(err) {
						t.Error("Expected Forbidden error not received")
					}
				case "CRD not found in cluster":
					if !apierrors.IsNotFound(err) {
						t.Error("Expected NotFound error not received")
					}
				}
			}
		})
	}
}
