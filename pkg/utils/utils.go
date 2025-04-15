package utils

import (
	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	FieldManager = "rbg"

	PatchAll    PatchType = "all"
	PatchSpec   PatchType = "spec"
	PatchStatus PatchType = "status"
)

type PatchType string

func PatchObjectApplyConfiguration(ctx context.Context, k8sClient client.Client, objApplyConfig interface{}, patchType PatchType) error {
	logger := log.FromContext(ctx)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(objApplyConfig)
	if err != nil {
		logger.Error(err, "Converting obj apply configuration to json.")
		return err
	}

	patch := &unstructured.Unstructured{
		Object: obj,
	}
	// Use server side apply and add fieldmanager to the rbg owned fields
	// If there are conflicts in the fields owned by the rbg controller, rbg will obtain the ownership and force override
	// these fields to the ones desired by the rbg controller
	// TODO b/316776287 add E2E test for SSA
	if patchType == PatchSpec || patchType == PatchAll {
		err = k8sClient.Patch(ctx, patch, client.Apply, &client.PatchOptions{
			FieldManager: FieldManager,
			Force:        ptr.To[bool](true),
		})
		if err != nil {
			logger.Error(err, "Using server side apply to patch object")
			return err
		}
	}

	if patchType == PatchStatus || patchType == PatchAll {
		err = k8sClient.Status().Patch(ctx, patch, client.Apply,
			&client.SubResourcePatchOptions{
				PatchOptions: client.PatchOptions{
					FieldManager: FieldManager,
					Force:        ptr.To[bool](true),
				},
			})
		if err != nil {
			logger.Error(err, "Using server side apply to patch object status")
			return err
		}
	}

	return nil
}

func ContainsString(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
