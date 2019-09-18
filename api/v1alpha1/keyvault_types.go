/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeyvaultSpec defines the desired state of Keyvault
type KeyvaultSpec struct {
	// Name is the name of the Azure Keyvault.
	Name string `json:"name"`
	// Location of the resource group (e.g., eastus2 or "West US")
	Location string `json:"location"`
	// ResourceGroup contains the Keyvault.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// TenantID contains the Subscription. Is a GUID.
	TenantID string `json:"tenantId"`
}

// KeyvaultStatus defines the observed state of Keyvault
type KeyvaultStatus struct {
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=keyvaults,shortName=kv,categories=all

// Keyvault is the Schema for the keyvaults API
type Keyvault struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeyvaultSpec   `json:"spec,omitempty"`
	Status KeyvaultStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeyvaultList contains a list of Keyvault
type KeyvaultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Keyvault `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Keyvault{}, &KeyvaultList{})
}
