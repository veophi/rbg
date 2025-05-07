package utils

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCheckOwnerReference(t *testing.T) {
	// Define target GVK for RoleBasedGroup
	targetGVK := schema.GroupVersionKind{
		Group:   "workloads.x-k8s.io",
		Version: "v1alpha1",
		Kind:    "RoleBasedGroup",
	}

	tests := []struct {
		name        string
		refs        []metav1.OwnerReference
		targetGVK   schema.GroupVersionKind // Optional override
		expected    bool
		description string
	}{
		// --------------------------
		// Basic Scenarios
		// --------------------------
		{
			name:        "empty owner references",
			refs:        []metav1.OwnerReference{},
			expected:    false,
			description: "Should return false for empty list",
		},
		{
			name: "no matching references",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
			},
			expected:    false,
			description: "Should ignore completely unrelated OwnerReferences",
		},
		{
			name: "exact match",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "workloads.x-k8s.io/v1alpha1",
					Kind:       "RoleBasedGroup",
				},
			},
			expected:    true,
			description: "Should return true for exact GVK match",
		},

		// --------------------------
		// Partial Match Scenarios
		// --------------------------
		{
			name: "group mismatch",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "wrong-group/v1alpha1",
					Kind:       "RoleBasedGroup",
				},
			},
			expected:    false,
			description: "Should fail when Group mismatches",
		},
		{
			name: "version mismatch",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "workloads.x-k8s.io/v1beta1",
					Kind:       "RoleBasedGroup",
				},
			},
			expected:    false,
			description: "Should fail when Version mismatches",
		},
		{
			name: "kind mismatch",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "workloads.x-k8s.io/v1alpha1",
					Kind:       "WrongKind",
				},
			},
			expected:    false,
			description: "Should fail when Kind mismatches",
		},

		// --------------------------
		// Special Format Handling
		// --------------------------
		{
			name: "core group (no group)",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "v1", // Core group (Group="")
					Kind:       "Pod",
				},
			},
			targetGVK: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			expected:    true,
			description: "Should handle core group (empty Group) correctly",
		},
		{
			name: "invalid apiVersion format",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "workloads.x-k8s.io/v1alpha1/extra",
					Kind:       "RoleBasedGroup",
				},
			},
			expected:    false,
			description: "Should skip invalid APIVersion formats",
		},
		{
			name: "case-sensitive check",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "Workloads.X-K8S.Io/v1alpha1", // Case mismatch
					Kind:       "RoleBasedGroup",
				},
			},
			expected:    false,
			description: "GVK matching should be case-sensitive",
		},

		// --------------------------
		// Multiple Entries Scenarios
		// --------------------------
		{
			name: "multiple refs with match",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				{
					APIVersion: "workloads.x-k8s.io/v1alpha1",
					Kind:       "RoleBasedGroup", // Match here
				},
			},
			expected:    true,
			description: "Should find match in multiple OwnerReferences",
		},
		{
			name: "multiple refs without match",
			refs: []metav1.OwnerReference{
				{
					APIVersion: "batch/v1",
					Kind:       "Job",
				},
				{
					APIVersion: "wrong-group/v1alpha1",
					Kind:       "RoleBasedGroup",
				},
			},
			expected:    false,
			description: "Should return false when no matches exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use custom targetGVK if specified
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					currentTargetGVK := targetGVK
					if !tt.targetGVK.Empty() {
						currentTargetGVK = tt.targetGVK
					}

					actual := CheckOwnerReference(tt.refs, currentTargetGVK)
					if actual != tt.expected {
						t.Errorf(
							"Test Case: %s\nDetails: %s\nExpected: %v, Actual: %v",
							tt.name,
							tt.description,
							tt.expected,
							actual,
						)
					}
				})
			}
		})
	}
}
