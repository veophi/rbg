package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	workloadsv1alpha1 "sigs.k8s.io/rbgs/api/workloads/v1alpha1"
	"testing"
)

func TestCheckOwnerReference(t *testing.T) {
	trueValue := true
	owners := []metav1.OwnerReference{
		{
			Kind:       "RoleBasedGroup",
			APIVersion: "workloads.x-k8s.io/v1alpha1",
			Name:       "nginx-cluster",
			UID:        "90b43d54-7b75-4eb7-8048-8c102c3d464d",
			Controller: &trueValue,
		},
	}

	gvk := schema.FromAPIVersionAndKind(workloadsv1alpha1.GroupVersion.String(), "RoleBasedGroup")

	if CheckOwnerReference(owners, gvk) {
		t.Log("CheckOwnerReference: true")
	} else {
		t.Error("CheckOwnerReference: false")
	}
}
