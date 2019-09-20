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
	// FriendlyName is the name of the secret locally inside the kubernetes object (a key in the map[][])
	FriendlyName *string `json:"friendlyName,omitempty"`
	// Location is the Azure location of the resource group (e.g., eastus2 or "West US").
	// Only required if Vault does not exist.
	// Must be used it conjuction with ResourceGroup and SubscriptionID
	Location *string `json:"location,omitempty"`
	// ResourceGroup contains the Keyvault.
	// Only required if Vault does not exist.
	// Must be used it conjuction with Location and SubscriptionID.
	ResourceGroup *string `json:"resourceGroup,omitempty"`
	// SubscriptionID contains the Resource group. Is a GUID.
	// Only required if Vault does not exist.
	// Must be used it conjuction with Location and ResourceGroup.
	SubscriptionID *string `json:"subscriptionId,omitempty"`
}

type SecretIdentifier struct {
	// Name is the name the corresponding Keyvault Secret.
	Name string `json:"name"`
	// Vault is the name of the Keyvault where this secret should be stored.
	Vault string `json:"vault"`
	// +optional
	// Kind allows specification of formatting other than the raw bytes in Keyvault.
	Kind *string `json:"kind,omitempty"`
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
