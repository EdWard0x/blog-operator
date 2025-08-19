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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BlogSpec defines the desired state of Blog
type BlogSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of Blog. Edit blog_types.go to remove/update
	// +optional
	//Foo *string `json:"foo,omitempty"`

	//------------------------Custom------------------------------

	// BackendImage is the container image for the blog backend.
	// +kubebuilder:validation:Required
	BackendImage string `json:"backendImage"`

	// FrontendImage is the container image for the stable version of the blog frontend.
	// +kubebuilder:validation:Required
	FrontendImage string `json:"frontendImage"`

	// FrontendCanaryImage is the optional container image for the canary version of the blog frontend.
	// +kubebuilder:validation:Optional
	FrontendCanaryImage string `json:"frontendCanaryImage,omitempty"`

	// Domain for the blog application.
	// +kubebuilder:validation:Required
	Domain string `json:"domain"`

	// KibanaDomain for the Kibana dashboard.
	// +kubebuilder:validation:Required
	KibanaDomain string `json:"kibanaDomain"`

	// TLSSecretName is the name of the secret containing the TLS certificate for the domain.
	// This secret must exist in the istio-system namespace.
	// +kubebuilder:validation:Required
	TLSSecretName string `json:"tlsSecretName"`

	// ImagePullSecret is the name of the secret for pulling images from a private registry.
	// This secret must exist in the same namespace as the Blog resource.
	// +kubebuilder:validation:Required
	ImagePullSecret string `json:"imagePullSecret"`
}

// BlogStatus defines the observed state of Blog.
type BlogStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//------------------------Custom------------------------------
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Blog is the Schema for the blogs API
type Blog struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Blog
	// +required
	Spec BlogSpec `json:"spec,omitempty"`

	// status defines the observed state of Blog
	// +optional
	Status BlogStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// BlogList contains a list of Blog
type BlogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Blog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Blog{}, &BlogList{})
}
