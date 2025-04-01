package utils

import (
	"context"
	"encoding/json"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdate(ctx context.Context, k8sClient client.Client, obj client.Object) error {
	existing := obj.DeepCopyObject().(client.Object)
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), existing)
	if err != nil && apierrors.IsNotFound(err) {
		return k8sClient.Create(ctx, obj)
	} else if err != nil {
		return err
	}
	// TODO: check invalidate update
	if !reflect.DeepEqual(obj, existing) {
		return k8sClient.Update(ctx, obj)
	}
	return nil
}

func MergeMap(target map[string]string, sources ...map[string]string) {
	if target == nil {
		return
	}
	for _, src := range sources {
		for k, v := range src {
			target[k] = v
		}
	}
}

func PrettyJson(obj interface{}) string {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}
