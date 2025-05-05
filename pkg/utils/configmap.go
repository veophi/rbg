package utils

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SemanticallyEqualConfigmap(old, new *corev1.ConfigMap) (bool, string) {
	if old == nil && new == nil {
		return true, ""
	}
	if old == nil || new == nil {
		return false, fmt.Sprintf("nil mismatch: old=%v, new=%v", old, new)
	}
	// Defensive copy to prevent side effects
	oldCopy := old.DeepCopy()
	newCopy := new.DeepCopy()

	oldCopy.Annotations = FilterSystemAnnotations(oldCopy.Annotations)
	newCopy.Annotations = FilterSystemAnnotations(newCopy.Annotations)

	objectMetaIgnoreOpts := cmpopts.IgnoreFields(
		metav1.ObjectMeta{},
		"ResourceVersion",
		"UID",
		"CreationTimestamp",
		"Generation",
		"ManagedFields",
		"SelfLink",
	)

	opts := cmp.Options{
		objectMetaIgnoreOpts,
		cmpopts.SortSlices(func(a, b metav1.OwnerReference) bool {
			return a.UID < b.UID // Make OwnerReferences comparison order-insensitive
		}),
		cmpopts.EquateEmpty(),
	}

	diff := cmp.Diff(oldCopy, newCopy, opts)
	return diff == "", diff
}

// filterSystemAnnotations 过滤系统注解
func FilterSystemAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}

	filtered := make(map[string]string)
	for k, v := range annotations {
		// 忽略 kubernetes.io/ 开头的系统注解
		if !strings.HasPrefix(k, "deployment.kubernetes.io/revision") &&
			!strings.HasPrefix(k, "rolebasedgroup.workloads.x-k8s.io/") {
			filtered[k] = v
		}
	}
	return filtered
}
