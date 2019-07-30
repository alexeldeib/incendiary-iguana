/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretSpec defines the desired state of Secret
type SecretSpec struct {
	// Name is the name the corresponding Keyvault Secret.
	Name string `json:"name"`
	// Vault is the name of the Keyvault where this secret should be stored.
	Vault string `json:"vault"`
	// LocalName is the desired name of the target Kubernetes secret.
	// Defaults to Name if not specified.
	LocalName *string `json:"localName,omitempty"`
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

// SingleSecretStatus defines reusable status properties of a single secret for composability.
type SingleSecretStatus struct {
	// TODO(ace): distinguish meaning more clearly.
	// Exists is true when the secret exists in the remote Keyvault.
	Exists bool `json:"exists"`
	// Available is true when the secret is ready for use in Kubernetes.
	Available bool `json:"available"`
	// LastKnownName is the name of this secret as seen when it was last reconciled.
	// This is useful for knowing when to delete/recreate a secret.
	LastKnownName string `json:"lastKnownName"`
}

// SecretStatus defines the observed state of Secret
type SecretStatus struct {
	// SingleSecretStatus has the actual status of the secret. Embedded for reuse with SecretBundle
	SingleSecretStatus `json:",inline"`
	// Generation is the last reconciled generation.
	Generation int64 `json:"generation"`
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
