/*
Copyright 2025.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudflareSpec defines the desired state of Cloudflare.
type CloudflareSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Cloudflare. Edit cloudflare_types.go to remove/update
	// Foo string `json:"foo,omitempty"`

	//+kubebuilder:validation:Required

	Ingress []IngressRule `json:"ingress,omitempty"`

	//+kubebuilder:validation:Required
	// +kubebuilder:default=1

	// Replicas int32 `json:"replicas,omitempty"`

	//+kubebuilder:validation:Required

	TunnelName string `json:"tunnel_name"`

	//+kubebuilder:validation:Required
	// +kubebuilder:default=1

	Replicas int32 `json:"replicas,omitempty"`
}
type IngressRule struct {
	//+kubebuilder:validation:Required

	Hostname string `json:"hostname"`

	//+kubebuilder:validation:Required

	Service string `json:"service"`
}

// CloudflareStatus defines the observed state of Cloudflare.
type CloudflareStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

const (
	TypeCloudflareViewAvailable = "Available"
	TypeCloudflareViewDegraded  = "Degraded"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Cloudflare is the Schema for the cloudflares API.
type Cloudflare struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudflareSpec   `json:"spec,omitempty"`
	Status CloudflareStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudflareList contains a list of Cloudflare.
type CloudflareList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cloudflare `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cloudflare{}, &CloudflareList{})
}
