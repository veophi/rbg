package utils

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckCrdExists checks if a Custom Resource Definition (CRD) with the specified name exists and is established.
func CheckCrdExists(reader client.Reader, crdName string) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	ctx := context.Background()
	if err := reader.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
		return fmt.Errorf("CRD %s not found: %v", crdName, err)
	}

	// Check Established status
	for _, cond := range crd.Status.Conditions {
		if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
			return nil
		}
	}
	return fmt.Errorf("CRD %s exists but not established", crdName)
}
