package utils

import (
	"context"
	"reflect"

	"k8s.io/cri-api/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdate(k8sClient client.Client, ctx context.Context, obj client.Object) error {
	existing := obj.DeepCopyObject().(client.Object)
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), existing)
	if err != nil && errors.IsNotFound(err) {
		return k8sClient.Create(ctx, obj)
	} else if err != nil {
		return err
	}

	if !reflect.DeepEqual(obj, existing) {
		return k8sClient.Update(ctx, obj)
	}
	return nil
}
