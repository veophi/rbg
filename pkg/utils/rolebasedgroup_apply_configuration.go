package utils

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metaapplyv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"sigs.k8s.io/rbgs/api/workloads/v1alpha1"
)

type RbgApplyConfiguration struct {
	metaapplyv1.TypeMetaApplyConfiguration    `json:",inline"`
	*metaapplyv1.ObjectMetaApplyConfiguration `json:"metadata,omitempty"`

	Status *RbgStatusApplyConfiguration `json:"status,omitempty"`
}

func RoleBasedGroup(name, namespace, kind, apiVersion string) *RbgApplyConfiguration {
	b := &RbgApplyConfiguration{}
	b.WithName(name)
	b.WithNamespace(namespace)
	b.WithKind(kind)
	b.WithAPIVersion(apiVersion)
	return b
}

func (b *RbgApplyConfiguration) WithAPIVersion(value string) *RbgApplyConfiguration {
	b.TypeMetaApplyConfiguration.APIVersion = &value
	return b
}

func (b *RbgApplyConfiguration) WithKind(value string) *RbgApplyConfiguration {
	b.TypeMetaApplyConfiguration.Kind = &value
	return b
}

func (b *RbgApplyConfiguration) WithNamespace(value string) *RbgApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.Namespace = &value
	return b
}

func (b *RbgApplyConfiguration) WithName(value string) *RbgApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.Name = &value
	return b
}

func (b *RbgApplyConfiguration) WithStatus(value *RbgStatusApplyConfiguration) *RbgApplyConfiguration {
	b.Status = value
	return b
}

func (b *RbgApplyConfiguration) ensureObjectMetaApplyConfigurationExists() {
	if b.ObjectMetaApplyConfiguration == nil {
		b.ObjectMetaApplyConfiguration = &metaapplyv1.ObjectMetaApplyConfiguration{}
	}
}

type RbgStatusApplyConfiguration struct {
	Conditions   []v1.Condition        `json:"conditions,omitempty"`
	RoleStatuses []v1alpha1.RoleStatus `json:"roleStatuses,omitempty"`
}

func RbgStatus() *RbgStatusApplyConfiguration {
	return &RbgStatusApplyConfiguration{}
}

func (b *RbgStatusApplyConfiguration) WithConditions(conditions ...v1.Condition) *RbgStatusApplyConfiguration {
	b.Conditions = append(b.Conditions, conditions...)
	return b
}

func (b *RbgStatusApplyConfiguration) WithRoleStatuses(values ...v1alpha1.RoleStatus) *RbgStatusApplyConfiguration {
	for i := range values {
		found := false
		for j, oldStatus := range b.RoleStatuses {
			// if found, update
			if values[i].Name == oldStatus.Name {
				found = true
				if values[i].Replicas != oldStatus.Replicas || values[i].ReadyReplicas != oldStatus.ReadyReplicas {
					b.RoleStatuses[j] = values[i]
				}
				break
			}
		}
		if !found {
			b.RoleStatuses = append(b.RoleStatuses, values[i])
		}
	}

	return b
}
