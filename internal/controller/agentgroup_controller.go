/*
Copyright 2025 LoongCollector Sigs.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"github.com/infraflows/loongcollector-operator/internal/emus"
	"github.com/infraflows/loongcollector-operator/internal/pkg/configserver"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/infraflows/loongcollector-operator/api/v1alpha1"
)

// AgentGroupReconciler reconciles a AgentGroup object
type AgentGroupReconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Event   record.EventRecorder
	BaseURL string
}

const (
	agentGroupFinalizer = "agentgroup.finalizers.infraflow.co"
)

// +kubebuilder:rbac:groups=infraflow.co,resources=agentgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infraflow.co,resources=agentgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infraflow.co,resources=agentgroups/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AgentGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("agentgroup", req.NamespacedName)

	if err := r.getConfigServerURL(ctx); err != nil {
		log.Error(err, "Failed to get ConfigServer URL")
		return reconcile.Result{}, err
	}

	agentGroup := &v1alpha1.AgentGroup{}
	err := r.Get(ctx, req.NamespacedName, agentGroup)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !agentGroup.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(agentGroup, agentGroupFinalizer) {
			if err := r.cleanupAgentGroup(ctx, agentGroup); err != nil {
				log.Error(err, "Failed to cleanup agent group")
				return reconcile.Result{}, err
			}

			controllerutil.RemoveFinalizer(agentGroup, agentGroupFinalizer)
			if err := r.Update(ctx, agentGroup); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(agentGroup, agentGroupFinalizer) {
		controllerutil.AddFinalizer(agentGroup, agentGroupFinalizer)
		if err := r.Update(ctx, agentGroup); err != nil {
			log.Error(err, "Failed to add finalizer")
			return reconcile.Result{}, err
		}
	}

	agentClient := configserver.NewConfigServerClient(r.BaseURL, &r.Client, agentGroup.Namespace)
	group := &configserver.AgentGroup{
		Name:        agentGroup.Spec.Name,
		Description: agentGroup.Spec.Description,
		Tags:        agentGroup.Spec.Tags,
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		// Try to create the agent group
		if err := agentClient.CreateAgentGroup(ctx, group); err != nil {
			// If the group already exists, try to update it
			if err := agentClient.UpdateAgentGroup(ctx, group); err != nil {
				lastErr = err
				log.Error(err, "Failed to create/update agent group, retrying", "attempt", i+1, "maxAttempts", maxRetries)
				time.Sleep(retryDelay)
				continue
			}
		}

		// Apply configs to the agent group
		for _, configName := range agentGroup.Spec.Configs {
			if err := agentClient.ApplyConfigToAgentGroup(ctx, configName, agentGroup.Spec.Name); err != nil {
				lastErr = err
				log.Error(err, "Failed to apply config to agent group, retrying", "config", configName, "attempt", i+1, "maxAttempts", maxRetries)
				time.Sleep(retryDelay)
				continue
			}
		}

		lastErr = nil
		break
	}

	if lastErr != nil {
		log.Error(lastErr, "Failed to manage agent group after retries")
		agentGroup.Status.Success = false
		agentGroup.Status.Message = emus.AgentGroupStatusFailed
		r.Event.Event(agentGroup, corev1.EventTypeWarning, "FailedToManageAgentGroup", lastErr.Error())
		agentGroup.Status.LastUpdateTime = metav1.Now()
		_ = r.Status().Update(ctx, agentGroup)
		return reconcile.Result{}, lastErr
	}

	agentGroup.Status.Success = true
	agentGroup.Status.Message = emus.AgentGroupStatusSuccess
	agentGroup.Status.AppliedConfigs = agentGroup.Spec.Configs
	r.Event.Event(agentGroup, corev1.EventTypeNormal, "SuccessfulManageAgentGroup", agentGroup.Status.Message)
	agentGroup.Status.LastUpdateTime = metav1.Now()
	if err := r.Status().Update(ctx, agentGroup); err != nil {
		log.Error(err, "Failed to update agent group status")
		return ctrl.Result{RequeueAfter: syncInterval}, err
	}

	log.Info("Successfully managed agent group")
	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// getConfigServerURL gets the ConfigServer URL from ConfigMap
func (r *AgentGroupReconciler) getConfigServerURL(ctx context.Context) error {
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Namespace: configMapNamespace, Name: configMapName}, configMap)
	if err != nil {
		if errors.IsNotFound(err) {
			r.BaseURL = defaultBaseURL
			return nil
		}
		return err
	}

	if url, ok := configMap.Data[configMapKey]; ok && url != "" {
		r.BaseURL = url
	} else {
		r.BaseURL = defaultBaseURL
	}
	return nil
}

// cleanupAgentGroup 清理AgentGroup相关的资源
func (r *AgentGroupReconciler) cleanupAgentGroup(ctx context.Context, agentGroup *v1alpha1.AgentGroup) error {
	log := r.Log.WithValues("agentgroup", agentGroup.Name)

	agentClient := configserver.NewConfigServerClient(r.BaseURL, &r.Client, agentGroup.Namespace)
	if err := agentClient.DeleteAgentGroup(ctx, agentGroup.Spec.Name); err != nil {
		log.Error(err, "Failed to delete agent group from config server")
		return err
	}

	log.Info("Successfully cleaned up agent group")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AgentGroup{}).
		Complete(r)
}
