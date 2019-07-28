/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretBundleSpec defines the desired state of SecretBundle
type SecretBundleSpec struct {
	// Name is the name the corresponding Keyvault Secret.
	Name string `json:"name"`
	// Secrets is a list of references to Keyvault secrets to sync to a single Kubernetes secret.
	// The keys in the map will be the keys in the Kubernetes secret.
	Secrets []SecretSpec `json:"secrets"`
}

// SecretBundleStatus defines the observed state of SecretBundle
type SecretBundleStatus struct {
	// Secrets is map of named statuses for individual secrets.
	Secrets map[string]SingleSecretStatus `json:"secrets"`
	// Generation is the last reconciled generation.
	Generation int64 `json:"generation"`
	// Desired is len(spec.Secrets): it is the number of configured secrets in this object.
	Desired int `json:"desired"`
	// Available is the number of desired secrets from Keyvault which were found.
	Available int `json:"available"`
	// Ready is the number of keys available for use in the target Kubernetes secret.
	// status.Desired == status.Ready implies an application depending on all of these secrets
	// could immediately begin using them.
	Ready int `json:"ready"`
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

// SecretBundleList contains a list of SecretBundle
type SecretBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretBundle{}, &SecretBundleList{})
}
