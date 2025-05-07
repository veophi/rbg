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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/rbgs/pkg/utils"
)

func TestRoleBasedGroupReconciler_CheckCrdExists(t *testing.T) {
	// Initialize test scheme with required types
	testScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = apiextensionsv1.AddToScheme(testScheme)

	// Target CRD name according to Kubernetes naming convention
	targetCRDName := "rolebasedgroups.workloads.x-k8s.io"

	type fields struct {
		apiReader client.Reader
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

func Test_hasValidOwnerRef(t *testing.T) {
	// Define the specific target GVK
	targetGVK := schema.GroupVersionKind{
		Group:   "workloads.x-k8s.io",
		Version: "v1alpha1",
		Kind:    "RoleBasedGroup",
	}

	type args struct {
		obj       client.Object
		targetGVK schema.GroupVersionKind
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// --------------------------
		// Basic Scenarios
		// --------------------------
		{
			name: "no owner references",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: nil, // Explicit nil
					},
				},
				targetGVK: targetGVK,
			},
			want: false,
		},
		{
			name: "empty owner references",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{}, // Empty slice
					},
				},
				targetGVK: targetGVK,
			},
			want: false,
		},
		{
			name: "owner reference exists but does not match",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "apps/v1",    // Group mismatch
								Kind:       "Deployment", // Kind mismatch
							},
						},
					},
				},
				targetGVK: targetGVK,
			},
			want: false,
		},
		{
			name: "owner reference matches exactly",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "workloads.x-k8s.io/v1alpha1", // Group/Version matches
								Kind:       "RoleBasedGroup",              // Kind matches
							},
						},
					},
				},
				targetGVK: targetGVK,
			},
			want: true,
		},

		// --------------------------
		// Edge Cases
		// --------------------------
		{
			name: "partial match in multiple owner references",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "batch/v1",
								Kind:       "Job",
							},
							{
								APIVersion: "workloads.x-k8s.io/v1alpha1", // Matches
								Kind:       "RoleBasedGroup",
							},
						},
					},
				},
				targetGVK: targetGVK,
			},
			want: true, // Returns true if at least one matches
		},
		{
			name: "correct group/version but wrong kind",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "workloads.x-k8s.io/v1alpha1",
								Kind:       "WrongKind", // Kind mismatch
							},
						},
					},
				},
				targetGVK: targetGVK,
			},
			want: false,
		},
		{
			name: "correct kind but wrong group",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "wrong-group/v1alpha1", // Group mismatch
								Kind:       "RoleBasedGroup",       // Kind matches
							},
						},
					},
				},
				targetGVK: targetGVK,
			},
			want: false,
		},
		{
			name: "invalid apiVersion format",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "workloads.x-k8s.io/v1alpha1/beta", // Invalid format
								Kind:       "RoleBasedGroup",
							},
						},
					},
				},
				targetGVK: targetGVK,
			},
			want: false, // Parsing fails → treated as non-match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasValidOwnerRef(tt.args.obj, tt.args.targetGVK); got != tt.want {
				t.Errorf("hasValidOwnerRef() = %v, want %v", got, tt.want)
			}
		})
	}
}
