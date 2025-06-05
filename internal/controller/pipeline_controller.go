package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/infraflows/loongcollector-operator/internal/emus"
	"github.com/infraflows/loongcollector-operator/internal/pkg/configserver"
	"github.com/infraflows/loongcollector-operator/internal/pkg/kube"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/infraflows/loongcollector-operator/api/v1alpha1"
)

// PipelineReconciler reconciles a Pipeline object
type PipelineReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Event    record.EventRecorder
	BaseURL  string
	informer *kube.PipelineInformer
}

const (
	defaultBaseURL     = "http://config-server:8899"
	configMapName      = "config-server-config"
	configMapNamespace = "loongcollector-system"
	configMapKey       = "configServerURL"
	maxRetries         = 3
	retryDelay         = time.Second * 5
	pipelineFinalizer  = "pipeline.finalizers.infraflow.co"
	syncInterval       = time.Minute * 5
)

// +kubebuilder:rbac:groups=infraflow.co,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infraflow.co,resources=pipelines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infraflow.co,resources=pipelines/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *PipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("pipeline", req.NamespacedName)

	pipeline := &v1alpha1.Pipeline{}
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		if errors.IsNotFound(err) {
			log.Info("pipeline resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to fetch pipeline resource")
		return ctrl.Result{}, err
	}
	if pipeline.DeletionTimestamp != nil {
		err := kube.HandleFinalizerWithCleanup(ctx, r.Client, pipeline, pipelineFinalizer, r.Log, r.cleanupPipeline)
		return ctrl.Result{}, err
	}

	return r.handlePipelineCreateOrUpdate(ctx, pipeline)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	dynClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	r.informer = kube.NewPipelineInformer(r.Client, dynClient, r.Log)
	go func() {
		if err := r.informer.Run(context.Background()); err != nil {
			r.Log.Error(err, "Failed to run pipeline informer")
		}
	}()

	// 在启动时处理所有现有的 Pipeline CRD
	go func() {
		// 等待 informer 缓存同步
		time.Sleep(5 * time.Second)

		var pipelines v1alpha1.PipelineList
		if err := r.List(context.Background(), &pipelines); err != nil {
			r.Log.Error(err, "Failed to list existing pipelines")
			return
		}

		for _, pipeline := range pipelines.Items {
			pipelineCopy := pipeline
			if _, err := r.handlePipelineCreateOrUpdate(context.Background(), &pipelineCopy); err != nil {
				r.Log.Error(err, "Failed to apply existing pipeline", "pipeline", pipeline.Name)
			}
		}
	}()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Pipeline{}).
		Complete(r)
}

// handlePipelineCreateOrUpdate 处理Pipeline创建或更新
func (r *PipelineReconciler) handlePipelineCreateOrUpdate(ctx context.Context, pipeline *v1alpha1.Pipeline) (ctrl.Result, error) {
	if err := kube.HandleFinalizerWithCleanup(ctx, r.Client, pipeline, pipelineFinalizer, r.Log, r.cleanupPipeline); err != nil {
		return ctrl.Result{}, err
	}

	if !r.shouldUpdatePipeline(ctx, pipeline) {
		r.Log.V(1).Info("Pipeline content unchanged, skipping update", "pipeline", pipeline.Name)
		return ctrl.Result{RequeueAfter: syncInterval}, nil
	}

	if err := r.applyPipeline(ctx, pipeline); err != nil {
		return r.updateStatusFailure(ctx, pipeline, emus.PipelineStatusFailed, err)
	}

	pipeline.Status.Success = true
	pipeline.Status.Message = emus.PipelineStatusSuccess
	pipeline.Status.LastUpdateTime = metav1.Now()
	pipeline.Status.LastAppliedConfig = v1alpha1.LastAppliedConfig{
		AppliedTime: metav1.Now(),
		Content:     pipeline.Spec.Content,
	}
	if err := r.Status().Update(ctx, pipeline); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// shouldUpdatePipeline 检查Pipeline是否需要更新
func (r *PipelineReconciler) shouldUpdatePipeline(ctx context.Context, pipeline *v1alpha1.Pipeline) bool {
	if pipeline.Status.LastAppliedConfig.Content == "" {
		return true
	}

	if pipeline.Spec.Content != pipeline.Status.LastAppliedConfig.Content {
		return true
	}

	if pipeline.Spec.AgentGroup != "" {
		// 获取当前AgentGroup的配置
		configServerClient := configserver.NewConfigServerClient(r.BaseURL, &r.Client, pipeline.Namespace)
		groups, err := configServerClient.ListAgentGroups(ctx)
		if err != nil {
			r.Log.Error(err, "Failed to list agent groups, assuming update needed", "pipeline", pipeline.Name)
			return true
		}

		// 检查AgentGroup是否存在
		groupExists := false
		for _, group := range groups {
			if group.Name == pipeline.Spec.AgentGroup {
				// TODO: 这里需要检查Pipeline是否在group中的逻辑
				groupExists = true
				// 检查Pipeline是否已经在group中
				for _, config := range group.Configs {
					if config == pipeline.Spec.Name {
						return false
					}
				}
				return true
			}
		}

		// 如果AgentGroup不存在，需要创建
		if !groupExists {
			return true
		}
	}

	return false
}
func (r *PipelineReconciler) applyPipeline(ctx context.Context, pipeline *v1alpha1.Pipeline) error {
	if err := r.getConfigServerURL(ctx); err != nil {
		return err
	}

	client := configserver.NewConfigServerClient(r.BaseURL, &r.Client, pipeline.Namespace)

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := r.tryApplyPipeline(ctx, client, pipeline); err != nil {
			lastErr = err
			time.Sleep(retryDelay)
			continue
		}
		return nil
	}
	return lastErr
}

// tryApplyPipeline 应用Pipeline重试
func (r *PipelineReconciler) tryApplyPipeline(ctx context.Context, client *configserver.ConfigServerClient, pipeline *v1alpha1.Pipeline) error {
	if err := client.CreateConfig(ctx, pipeline); err != nil {
		return err
	}

	agentGroup := pipeline.Spec.AgentGroup
	if agentGroup == "" {
		return nil
	}

	// 获取已有AgentGroup列表
	groups, err := client.ListAgentGroups(ctx)
	if err != nil {
		return err
	}

	// 判断是否存在
	exists := false
	for _, group := range groups {
		if group.Name == agentGroup {
			exists = true
			break
		}
	}

	// 创建AgentGroup（如果不存在）
	if !exists {
		newGroup := &configserver.AgentGroup{
			Name:        agentGroup,
			Description: "Created automatically for pipeline " + pipeline.Spec.Name,
		}
		if err := client.CreateAgentGroup(ctx, newGroup); err != nil {
			return err
		}
	}

	// 关联Pipeline到AgentGroup
	return client.ApplyConfigToAgentGroup(ctx, pipeline.Spec.Name, agentGroup)
}

// updateStatusFailure 更新Pipeline状态为失败
func (r *PipelineReconciler) updateStatusFailure(ctx context.Context, pipeline *v1alpha1.Pipeline, msg string, err error) (ctrl.Result, error) {
	pipeline.Status.Success = false
	pipeline.Status.Message = msg
	r.Event.Event(pipeline, corev1.EventTypeWarning, msg, err.Error())
	pipeline.Status.LastUpdateTime = metav1.Now()
	_ = r.Status().Update(ctx, pipeline)
	return ctrl.Result{}, err
}

// getConfigServerURL gets the ConfigServer URL from ConfigMap
func (r *PipelineReconciler) getConfigServerURL(ctx context.Context) error {
	configMap, err := kube.GetConfigMapByLabel(ctx, r.Client, configMapName,
		configMapNamespace, map[string]string{
			"app": "config-server",
		})
	if err != nil {
		return err
	}

	if url, ok := configMap.Data[configMapKey]; ok && url != "" {
		r.BaseURL = url
	} else {
		r.BaseURL = defaultBaseURL
	}
	return nil
}

// cleanupPipeline 清理Pipeline相关的资源
func (r *PipelineReconciler) cleanupPipeline(ctx context.Context, pipeline *v1alpha1.Pipeline) error {
	log := r.Log.WithValues("pipeline", pipeline.Name)

	// 如果指定了AgentGroup，从AgentGroup中移除Pipeline
	if pipeline.Spec.AgentGroup != "" {
		configServerClient := configserver.NewConfigServerClient(r.BaseURL, &r.Client, pipeline.Namespace)
		if err := configServerClient.RemoveConfigFromAgentGroup(ctx, pipeline.Spec.Name, pipeline.Spec.AgentGroup); err != nil {
			log.Error(err, "Failed to remove pipeline from agent group")
			return err
		}
	}

	configServerClient := configserver.NewConfigServerClient(r.BaseURL, &r.Client, pipeline.Namespace)
	if err := configServerClient.DeleteConfig(ctx, pipeline.Spec.Name); err != nil {
		log.Error(err, "Failed to delete pipeline from agent")
		return err
	}

	log.Info("Successfully cleaned up pipeline from agent")
	return nil
}
