package kube

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// HandleFinalizerWithCleanup handle finalizer with cleanup
func HandleFinalizerWithCleanup[T client.Object](
	ctx context.Context, c client.Client,
	obj T, finalizer string,
	log logr.Logger, cleanupFn func(context.Context, T) error) error {

	if reflect.ValueOf(obj).IsNil() {
		return fmt.Errorf("object is nil")
	}

	if obj.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(obj, finalizer) {
			if err := cleanupFn(ctx, obj); err != nil {
				return err
			}
			controllerutil.RemoveFinalizer(obj, finalizer)
			return c.Update(ctx, obj)
		}
		return nil
	}

	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		controllerutil.AddFinalizer(obj, finalizer)
		return c.Update(ctx, obj)
	}

	return nil
}
