package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSemanticallyEqualConfigmap(t *testing.T) {
	tests := []struct {
		name     string
		oldCM    *corev1.ConfigMap
		newCM    *corev1.ConfigMap
		expected bool
	}{
		{
			name: "Name mismatch",
			oldCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-1",
					Namespace: "default",
				},
			},
			newCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-2",
					Namespace: "default",
				},
			},
			expected: false,
		},
		{
			name: "Namespace mismatch",
			oldCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-1",
					Namespace: "ns-1",
				},
			},
			newCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-1",
					Namespace: "ns-2",
				},
			},
			expected: false,
		},
		{
			name: "Filter system annotations - both have same user annotations",
			oldCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision": "2",
						"user-annotation":                   "value",
					},
				},
			},
			newCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"rolebasedgroup.workloads.x-k8s.io/hash": "abc123",
						"user-annotation":                        "value",
					},
				},
			},
			expected: true,
		},
		{
			name: "Filter system annotations - different user annotations",
			oldCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision": "1",
						"user-annotation":                   "old-value",
					},
				},
			},
			newCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision": "2",
						"user-annotation":                   "new-value",
					},
				},
			},
			expected: false,
		},
		{
			name: "Mixed system and user annotations",
			oldCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision":      "1",
						"rolebasedgroup.workloads.x-k8s.io/hash": "abc",
						"user":                                   "config",
					},
				},
			},
			newCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"user": "config",
					},
				},
			},
			expected: true,
		},
		{
			name: "Same name/namespace with data change",
			oldCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
				},
				Data: map[string]string{"key": "old"},
			},
			newCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
				},
				Data: map[string]string{"key": "new"},
			},
			expected: false,
		},
		{
			name: "Equal with different system annotations",
			oldCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision": "1",
					},
				},
			},
			newCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm",
					Namespace: "default",
					Annotations: map[string]string{
						"deployment.kubernetes.io/revision": "2",
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal, diff := SemanticallyEqualConfigmap(tt.oldCM, tt.newCM)
			if equal != tt.expected {
				t.Errorf("%s: Expected %v, got %v.\nDiff: %s",
					tt.name, tt.expected, equal, diff)
			}

			// Test symmetry
			if tt.oldCM != nil && tt.newCM != nil {
				reverseEqual, _ := SemanticallyEqualConfigmap(tt.newCM, tt.oldCM)
				if reverseEqual != equal {
					t.Errorf("%s: Asymmetric comparison! Forward=%v Reverse=%v",
						tt.name, equal, reverseEqual)
				}
			}
		})
	}
}
