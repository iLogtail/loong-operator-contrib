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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AgentGroupSpec defines the desired state of AgentGroup.
type AgentGroupSpec struct {
	// Name of the agent group
	Name string `json:"name"`
	// Description of the agent group
	Description string `json:"description,omitempty"`
	// Tags for the agent group
	Tags []string `json:"tags,omitempty"`
	// Configs that should be applied to this agent group
	Configs []string `json:"configs,omitempty"`
}

// AgentGroupStatus defines the observed state of AgentGroup.
type AgentGroupStatus struct {
	// Success indicates whether the agent group was successfully created
	Success bool `json:"success"`
	// Message is the message of the agent group
	Message string `json:"message,omitempty"`
	// LastUpdateTime is the last time the agent group was updated
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// AppliedConfigs is the list of configs that have been applied to this agent group
	AppliedConfigs []string `json:"appliedConfigs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AgentGroup is the Schema for the agentgroups API.
type AgentGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentGroupSpec   `json:"spec,omitempty"`
	Status AgentGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentGroupList contains a list of AgentGroup.
type AgentGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentGroup{}, &AgentGroupList{})
}
