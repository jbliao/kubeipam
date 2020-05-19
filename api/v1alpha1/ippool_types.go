/*


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

// IPPoolSpec defines the desired state of IPPool
type IPPoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Some field should be replaced with other [un]marshalable type after
	// this issue: https://github.com/kubernetes-sigs/controller-tools/issues/391
	// has been resolved

	// Allocations is the set of allocated IPs for the given range. Its` indices are a direct mapping to the
	// IP with the same index/offset for the pool's range.
	// +kubebuilder:validation:Optional
	Allocations []IPAllocation `json:"allocations"`

	// Addresses is the set of allocable ip address
	Addresses []string `json:"addresses"`

	// Type defined type of the external IPAM service to this IPPool
	Type string `json:"type"`

	// RawConfig is the driver specific configuration in raw json format
	RawConfig string `json:"rawConfig"`
}

// IPAllocation represents metadata about the pod/container owner of a specific IP
type IPAllocation struct {
	Address     string `json:"address"`
	ContainerID string `json:"id"`
}

// IPPoolStatus defines the observed state of IPPool
type IPPoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// IPPool is the Schema for the ippools API
type IPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPPoolSpec   `json:"spec,omitempty"`
	Status IPPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPPoolList contains a list of IPPool
type IPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPPool{}, &IPPoolList{})
}
