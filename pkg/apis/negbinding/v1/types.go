/*
Copyright 2025 The Kubernetes Authors.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// NetworkEndpointGroupBinding is a resource that binds a Kubernetes Service to a set of Network Endpoint Groups (NEGs).
type NetworkEndpointGroupBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkEndpointGroupBindingSpec   `json:"spec,omitempty"`
	Status NetworkEndpointGroupBindingStatus `json:"status,omitempty"`
}

// NetworkEndpointGroupBindingSpec defines the desired state of NetworkEndpointGroupBinding.
type NetworkEndpointGroupBindingSpec struct {
	// ServiceRef is a reference to the Kubernetes Service that this binding applies to.
	// The service must be in the same namespace as the binding.
	ServiceRef ServiceReference `json:"serviceRef"`
	// NetworkEndpointGroups is a list of references to the Network Endpoint Groups that should be associated with the service.
	NetworkEndpointGroups []NetworkEndpointGroupReference `json:"networkEndpointGroups"`
}

// ServiceReference is a reference to a Kubernetes Service.
type ServiceReference struct {
	// Name is the name of the service.
	Name string `json:"name"`
}

// NetworkEndpointGroupReference is a reference to a GCE Network Endpoint Group.
type NetworkEndpointGroupReference struct {
	// Name is the name of the Network Endpoint Group.
	Name string `json:"name"`
}

// NetworkEndpointGroupBindingStatus defines the observed state of NetworkEndpointGroupBinding.
type NetworkEndpointGroupBindingStatus struct {
	// Conditions represents the current state of the NetworkEndpointGroupBinding.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkEndpointGroupBindingList is a list of NetworkEndpointGroupBinding resources.
type NetworkEndpointGroupBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkEndpointGroupBinding `json:"items"`
}
