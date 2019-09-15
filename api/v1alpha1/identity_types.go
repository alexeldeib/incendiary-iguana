/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IdentitySpec defines the desired state of Identity
type IdentitySpec struct {
	// Name is the name of the resource.
	Name string `json:"name"`
	// Location of resource group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup contain the resource..
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID is the GUID of the Azure subscription containing the resoruce group.
	SubscriptionID string `json:"subscriptionId"`
}

// IdentityStatus defines the observed state of Identity
type IdentityStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
	// ObservedGeneration is the iteration of user-provided spec which has already been reconciled.
	// This is used to decide when to re-reconcile changes.
	ObservedGeneration int64 `json:"observedGeneration"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=resourcegroups,shortName=rg,categories=all

// Identity is the Schema for the resourcegroups API
type Identity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IdentitySpec   `json:"spec,omitempty"`
	Status IdentityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IdentityList contains a list of Identity
type IdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Identity `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Identity{}, &IdentityList{})
}
