//
// Copyright 2021 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PolicyControllerSpec defines the desired state of PolicyController
type PolicyControllerSpec struct {
	OperatorVersion string `json:"operatorVersion,omitempty"`
	Replicas        int32  `json:"replicas"`
	// ImageRegistry deprecated, define image in operator.yaml
	ImageRegistry string `json:"imageRegistry,omitempty"`
	// ImageTagPostfix deprecated, define image in operator.yaml
	ImageTagPostfix string                       `json:"imageTagPostfix,omitempty"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// PolicyControllerStatus defines the observed state of PolicyController
type PolicyControllerStatus struct {
	Nodes    []string      `json:"nodes"`
	Versions StatusVersion `json:"versions,omitempty"`
}

// StatusVersion defines the Operator versions
type StatusVersion struct {
	Reconciled string `json:"reconciled"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PolicyController is the Schema for the policycontrollers API
type PolicyController struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicyControllerSpec   `json:"spec,omitempty"`
	Status PolicyControllerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PolicyControllerList contains a list of PolicyController
type PolicyControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyController `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PolicyController{}, &PolicyControllerList{})
}
