/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretSpec defines the desired state of Secret
type SecretSpec struct {
	SecretIdentifier `json:",inline"`
	// TargetSecret is the name of the Kubernetes secret object this CRD should target
	TargetSecret string `json:"targetSecret,omitempty"`
	// TargetValue is the name of the key inside the kubernetes object (a key in the map[][])
	TargetValue string `json:"targetValue,omitempty"`
}

type SecretIdentifier struct {
	// Name is the name the corresponding Keyvault Secret.
	Name string `json:"name"`
	// Vault is the name of the Keyvault where this secret should be stored.
	Vault string `json:"vault"`
	// +optional
	// Kind allows specification of formatting other than the raw bytes in Keyvault.
	Kind *string `json:"kind,omitempty"`
	// If kind is x509 and reverse is true, this will fix the chain order.
	Reverse bool `json:"reverse,omitempty"`
}

// SecretStatus defines the observed state of Secret
type SecretStatus struct {
	State *string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Secret is the Schema for the secrets API
type Secret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretSpec   `json:"spec,omitempty"`
	Status SecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretList contains a list of Secret
type SecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Secret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Secret{}, &SecretList{})
}
