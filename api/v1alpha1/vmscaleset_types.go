/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VMScaleSetSpec defines the desired state of VMScaleSet
type VMScaleSetSpec struct {
	// Name is the name of the Azure resource group.
	Name string `json:"name"`
	// Location of the resource group (e.g., eastus2 or "West US")
	Location string `json:"location"`
	// SubscriptionID is the GUID of the subscription for this resource group.
	SubscriptionID string `json:"subscriptionId"`
}

// VMScaleSetStatus defines the observed state of VMScaleSet
type VMScaleSetStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=vmscaleset,shortName=vmss,categories=all

// VMScaleSet is the schema for the VMSS API
type VMScaleSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMScaleSetSpec   `json:"spec,omitempty"`
	Status VMScaleSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VMScaleSetList contains a list of VMScaleSet
type VMScaleSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMScaleSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMScaleSet{}, &VMScaleSetList{})
}
