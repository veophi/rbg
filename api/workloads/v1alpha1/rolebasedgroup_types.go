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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleBasedGroupSpec defines the desired state of RoleBasedGroup.
type RoleBasedGroupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Required
	Roles []RoleSpec `json:"roles"`
}

// RoleSpec defines the specification for a role in the group
type RoleSpec struct {
	// Unique identifier for the role
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas"`

	// Number of replicas for this role
	// +optional
	Dependencies []string `json:"dependencies,omitempty"`

	// Workload type specification
	// +kubebuilder:default={apiVersion:"apps/v1", kind:"StatefulSet"}
	// +optional
	Workload WorkloadSpec `json:"workload,omitempty"`

	// Pod template specification
	// +kubebuilder:validation:Required
	Template corev1.PodTemplateSpec `json:"template"`

	// +optional
	ServicePorts []corev1.ServicePort `json:"servicePorts,omitempty"`
}

type WorkloadSpec struct {
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/v[0-9]+((alpha|beta)[0-9]+)?$`
	// +kubebuilder:default="apps/v1"
	APIVersion string `json:"apiVersion"`

	// +optional
	// +kubebuilder:default="StatefulSet"
	Kind string `json:"kind"`
}

// RoleBasedGroupStatus defines the observed state of RoleBasedGroup.
type RoleBasedGroupStatus struct {
	// The generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions track the condition of the RBG
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Status of individual roles
	RoleStatuses []RoleStatus `json:"roleStatuses"`
}

// RoleStatus shows the current state of a specific role
type RoleStatus struct {
	// Name of the role
	Name string `json:"name"`

	// Number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas"`

	// Total number of desired replicas
	Replicas int32 `json:"replicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName={rbg}

// RoleBasedGroup is the Schema for the rolebasedgroups API.
type RoleBasedGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleBasedGroupSpec   `json:"spec,omitempty"`
	Status RoleBasedGroupStatus `json:"status,omitempty"`
}

type RoleBasedGroupConditionType string

// These are built-in conditions of a RBG.
const (
	// RoleBasedGroupAvailable means the rbg is available, ie, at least the
	// minimum available groups are up and running.
	RoleBasedGroupAvailable RoleBasedGroupConditionType = "Available"

	// RoleBasedGroupProgressing means rbg is progressing. Progress for a
	// rbg replica is considered when a new group is created, and when new pods
	// scale up and down. Before a group has all its pods ready, the group itself
	// will be in progressing state. And any group in progress will make
	// the rbg as progressing state.
	RoleBasedGroupProgressing RoleBasedGroupConditionType = "Progressing"

	// RoleBasedGroupUpdateInProgress means rbg is performing a rolling update. UpdateInProgress
	// is true when the rbg is in upgrade process after the (leader/worker) template is updated. If only replicas is modified, it will
	// not be considered as UpdateInProgress.
	RoleBasedGroupUpdateInProgress RoleBasedGroupConditionType = "UpdateInProgress"
)

// +kubebuilder:object:root=true

// RoleBasedGroupList contains a list of RoleBasedGroup.
type RoleBasedGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RoleBasedGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RoleBasedGroup{}, &RoleBasedGroupList{})
}
