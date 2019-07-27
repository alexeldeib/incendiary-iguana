/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceGroupSpec defines the desired state of ResourceGroup
type ResourceGroupSpec struct {
	// Name is the name of the Azure resource group.
	Name string `json:"name"`
	// Location of the resource group (e.g., eastus2 or "West US")
	Location string `json:"location"`
	// SubscriptionID is the GUID of the subscription for this resource group.
	SubscriptionID string `json:"subscriptionId"`
}

// ResourceGroupStatus defines the observed state of ResourceGroup
type ResourceGroupStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState string `json:"provisioningState"`
	Generation        int64  `json:"generation"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ResourceGroup is the Schema for the resourcegroups API
type ResourceGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceGroupSpec   `json:"spec,omitempty"`
	Status ResourceGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceGroupList contains a list of ResourceGroup
type ResourceGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceGroup{}, &ResourceGroupList{})
}
