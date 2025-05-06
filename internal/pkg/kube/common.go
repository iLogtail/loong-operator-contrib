package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetConfigMapByLabel 根据Label获取ConfigMap
func GetConfigMapByLabel(ctx context.Context, cli client.Client, name, namespace string, labelSelector map[string]string) (*corev1.ConfigMap, error) {
	var configMapList corev1.ConfigMapList
	
	if err := cli.List(ctx, &configMapList, client.InNamespace(namespace), client.MatchingLabels(labelSelector)); err != nil {
		return nil, fmt.Errorf("failed to list DaemonSets: %w", err)
	}

	if len(configMapList.Items) == 0 {
		return nil, fmt.Errorf("no ConfigMap found with labels: %v", labelSelector)
	}

	if len(configMapList.Items) > 1 {
		return nil, fmt.Errorf("multiple ConfigMaps found with labels: %v", labelSelector)
	}

	return &configMapList.Items[0], nil
}
