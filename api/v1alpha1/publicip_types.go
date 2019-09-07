/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PublicIPSpec defines the desired state of PublicIP
type PublicIPSpec struct {
	// Name is the name of the security group.
	Name string `json:"name"`
	// Location osecurity group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup containsecurity group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// SKU is either basic or standard, representing the SKU of the IP in Azure.
	SKU *string `json:"sku,omitempty"`
}

// PublicIPStatus defines the observed state of PublicIP
type PublicIPStatus struct {
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

// PublicIP is the Schema for the publicips API
type PublicIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PublicIPSpec   `json:"spec,omitempty"`
	Status PublicIPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PublicIPList contains a list of PublicIP
type PublicIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PublicIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PublicIP{}, &PublicIPList{})
}
