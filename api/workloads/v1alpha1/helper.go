package v1alpha1

import "fmt"

func (rbg *RoleBasedGroup) GetCommonLabelsFromRole(role *RoleSpec) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       rbg.Name,
		"app.kubernetes.io/component":  role.Name,
		"app.kubernetes.io/managed-by": ControllerName,
		"app.kubernetes.io/instance":   rbg.Name,
		SetNameLabelKey:                rbg.Name,
		SetRoleLabelKey:                role.Name,
	}
}

func (rbg *RoleBasedGroup) GetCommonAnnotationsFromRole(role *RoleSpec) map[string]string {
	return map[string]string{
		RoleSizeAnnotationKey: fmt.Sprintf("%d", *role.Replicas),
	}
}

func (rbg *RoleBasedGroup) GetWorkloadName(role *RoleSpec) string {
	return fmt.Sprintf("%s-%s", rbg.Name, role.Name)
}
