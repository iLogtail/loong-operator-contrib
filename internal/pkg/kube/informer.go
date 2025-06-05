package kube

import (
	"context"
	"fmt"

	"k8s.io/utils/clock"

	"github.com/go-logr/logr"
	"github.com/infraflows/loongcollector-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PipelineInformer struct {
	client    client.Client
	log       logr.Logger
	informer  cache.SharedIndexInformer
	workQueue workqueue.TypedRateLimitingInterface[string]
}

var pipelineGVR = schema.GroupVersionResource{
	Group:    "loongcollector.infraflows.co",
	Version:  "v1alpha1",
	Resource: "pipelines", // 👈 注意复数形式
}

// NewPipelineInformer 创建新的PipelineInformer实例
func NewPipelineInformer(client client.Client, dynClient dynamic.Interface, log logr.Logger) *PipelineInformer {
	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[string](), // 必须是 TypedRateLimiter[string]
		workqueue.TypedRateLimitingQueueConfig[string]{
			Name:  "PipelineInformer",
			Clock: clock.RealClock{},
			// MetricsProvider: metrics.NewPrometheusProvider(), // 可选
		},
	)

	informer := &PipelineInformer{
		client:    client,
		log:       log.WithName("pipeline-informer"),
		workQueue: queue,
	}

	// 创建SharedIndexInformer
	informer.informer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				list := &v1alpha1.PipelineList{}
				if err := client.List(context.TODO(), list); err != nil {
					return nil, err
				}
				return list, nil
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return dynClient.
					Resource(pipelineGVR).
					Namespace(metav1.NamespaceAll).
					Watch(context.TODO(), options)
			},
		},
		&v1alpha1.Pipeline{},
		0,
		cache.Indexers{},
	)

	return informer
}

// Run 启动Informer
func (p *PipelineInformer) Run(ctx context.Context) error {
	p.log.Info("Starting Pipeline informer")
	defer p.workQueue.ShutDown()

	// 添加事件处理器
	_, err := p.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			p.AddEvent(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			p.UpdateEvent(new)
		},
		DeleteFunc: func(obj interface{}) {
			p.DeleteEvent(obj)
		},
	})
	if err != nil {
		return err
	}

	// 启动Informer
	go p.informer.Run(ctx.Done())

	// 等待缓存同步
	if !cache.WaitForCacheSync(ctx.Done(), p.informer.HasSynced) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	p.log.Info("Pipeline informer started successfully")

	<-ctx.Done()
	return nil
}

// GetWorkQueue 获取工作队列
func (p *PipelineInformer) GetWorkQueue() workqueue.TypedRateLimitingInterface[string] {
	return p.workQueue
}

// GetInformer 获取SharedIndexInformer
func (p *PipelineInformer) GetInformer() cache.SharedIndexInformer {
	return p.informer
}

// AddEvent 处理添加事件
func (p *PipelineInformer) AddEvent(obj interface{}) {
	if key, err := cache.MetaNamespaceKeyFunc(obj); err != nil {
		p.log.Error(err, "Failed to get key from cache")
		return
	} else {
		p.workQueue.Add(key)
		p.log.Info("Added object to workQueue", "key", key)
	}
}

// UpdateEvent 处理更新事件
func (p *PipelineInformer) UpdateEvent(new interface{}) {
	if key, err := cache.MetaNamespaceKeyFunc(new); err != nil {
		p.log.Error(err, "Failed to get key from cache")
		return
	} else {
		p.workQueue.Add(key)
		p.log.Info("Updated object in workQueue", "key", key)
	}
}

// DeleteEvent 处理删除事件
func (p *PipelineInformer) DeleteEvent(obj interface{}) {
	if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err != nil {
		p.log.Error(err, "Failed to get key from cache")
		return
	} else {
		p.workQueue.Add(key)
		p.log.Info("Deleted object from workQueue", "key", key)
	}
}
