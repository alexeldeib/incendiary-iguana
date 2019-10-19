/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TLSSecretSpec defines the desired state of TLSSecret
type TLSSecretSpec struct {
	// Name is the name the corresponding Keyvault Secret.
	Name string `json:"name"`
	// Vault is the name of the Keyvault where this secret should be stored.
	Vault   string `json:"vault"`
	Reverse bool   `json:"reverse,omitempty"`
}

// TLSSecretStatus defines the observed state of TLSSecret
type TLSSecretStatus struct {
	State *string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TLSSecret is the Schema for the secrets API
type TLSSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TLSSecretSpec   `json:"spec,omitempty"`
	Status TLSSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TLSSecretList contains a list of TLSSecret
type TLSSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TLSSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TLSSecret{}, &TLSSecretList{})
}
