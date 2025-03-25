package v1

func (rbg *RoleBasedGroup) GetCommonLabelsFromRole(role *RoleSpec) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       rbg.Name,
		"app.kubernetes.io/component":  role.Name,
		"app.kubernetes.io/managed-by": ControllerName,
		"app.kubernetes.io/instance":   rbg.Name,
	}
}

func (rbg *RoleBasedGroup) GetCommonAnnotationsFromRole(role *RoleSpec) map[string]string {
	return map[string]string{}
}

func (rbg *RoleBasedGroup) GetRole(name string) (role *RoleSpec) {
	// return map[string]string{}
	for _, role := range rbg.Spec.Roles {
		if role.Name == name {
			return &role
		}
	}
	return
}
