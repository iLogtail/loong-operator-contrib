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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PipelineSpec defines the desired state of Pipeline.
type PipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name of the pipeline
	Name string `json:"name"`
	// content is the pipeline configuration
	Content string `json:"content"`

	// AgentGroup specifies the agent group to which this pipeline should be applied
	// +optional
	AgentGroup string `json:"agentGroup,omitempty"`

	// 支持logtail
	// https://help.aliyun.com/zh/sls/user-guide/recommend-use-aliyunpipelineconfig-to-manage-collection-configurations?spm=a2c4g.11186623.help-menu-28958.d_2_1_1_3_2_0.3b56694e44bSyR&scm=20140722.H_2833390._.OR_help-T_cn~zh-V_1#770941e164v6h
	// Project defines the SLS project configuration
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Project runtime.RawExtension `json:"project,omitempty"`
	// LogStores defines the SLS logstore configurations
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	LogStores runtime.RawExtension `json:"logStores,omitempty"`
	// MachineGroups defines the machine groups for log collection
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	MachineGroups runtime.RawExtension `json:"machineGroups,omitempty"`
	// EnableUpgradeOverride indicates whether to enable upgrade override
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	EnableUpgradeOverride bool `json:"enableUpgradeOverride,omitempty"`
}

// PipelineStatus defines the observed state of Pipeline.
type PipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Success indicates whether the pipeline was successfully created
	Success bool `json:"success"`
	// Message is the message of the pipeline
	Message string `json:"message,omitempty"`
	// LastUpdateTime is the last time the pipeline was updated
	LastUpdateTime metav1.Time `json:"LastUpdateTime,omitempty"`
	// LastAppliedConfig is the last applied configuration of the pipeline
	LastAppliedConfig LastAppliedConfig `json:"lastAppliedConfig,omitempty"`
}

type LastAppliedConfig struct {
	AppliedTime metav1.Time `json:"appliedTime,omitempty"`
	Content     string      `json:"content,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Pipeline is the Schema for the pipelines API.
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec,omitempty"`
	Status PipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineList contains a list of Pipeline.
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
