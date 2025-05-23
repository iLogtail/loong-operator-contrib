package kube

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ResourceHelper struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Recorder events.FakeRecorder
	Log      logr.Logger
	Mapper   meta.RESTMapper
}

func NewResourceHelper(mgr manager.Manager, log logr.Logger) *ResourceHelper {
	return &ResourceHelper{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    log.WithName("ResourceHelper"),
		Mapper: mgr.GetRESTMapper(),
	}
}

// CreateOrUpdateWithOwner 创建或更新资源，设置 ownerReference，自动识别 GVK
func (h *ResourceHelper) CreateOrUpdateWithOwner(ctx context.Context, owner client.Object, obj client.Object) error {
	log := h.Log.WithValues("namespace", obj.GetNamespace(), "name", obj.GetName())

	if err := controllerutil.SetControllerReference(owner, obj, h.Scheme); err != nil {
		log.Error(err, "failed to set controller reference")
		return err
	}

	gvk, err := h.getGVK(obj)
	if err != nil {
		log.Error(err, "failed to get GVK")
		return err
	}
	log = log.WithValues("gvk", gvk.String())

	key := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	existing := obj.DeepCopyObject().(client.Object)
	err = h.Client.Get(ctx, key, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating new resource")
			err = h.Client.Create(ctx, obj)
			if err != nil {
				log.Error(err, "failed to create resource")
			}
			return err
		}
		log.Error(err, "failed to get resource")
		return err
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	log.Info("Updating existing resource")
	err = h.Client.Update(ctx, obj)
	if err != nil {
		log.Error(err, "failed to update resource")
	}
	return err
}

// getGVK 返回对象的 GVK
func (h *ResourceHelper) getGVK(obj client.Object) (schema.GroupVersionKind, error) {
	gvks, _, err := h.Scheme.ObjectKinds(obj)
	if err != nil || len(gvks) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("failed to get GVK: %w", err)
	}
	return gvks[0], nil
}
