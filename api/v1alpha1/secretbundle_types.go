/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true

// SecretBundleSpec defines the desired state of SecretBundle
type SecretBundleSpec struct {
	// Name is the name the corresponding Keyvault Secret.
	Name string `json:"name"`
	// Secrets is a list of references to Keyvault secrets to sync to a single Kubernetes secret.
	// The keys in the map will be the keys in the Kubernetes secret.
	Secrets map[string]SecretIdentifier `json:"secrets"`
}

// SecretBundleStatus defines the observed state of SecretBundle
type SecretBundleStatus struct {
	// Secrets is map of named statuses for individual secrets.
	Secrets map[string]string `json:"secrets,omitempty"`
	State   *string           `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SecretBundle is the Schema for the secretbundles API
type SecretBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretBundleSpec   `json:"spec,omitempty"`
	Status SecretBundleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// SecretBundleList contains a list of SecretBundle
type SecretBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretBundle{}, &SecretBundleList{})
}
