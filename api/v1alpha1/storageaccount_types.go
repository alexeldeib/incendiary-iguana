/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageAccountSpec defines the desired state of StorageAccount
type StorageAccountSpec struct {
	// Name is the name of the resource.
	Name string `json:"name"`
	// Location of resource group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup containing the resource.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// TargetSecret +optional
	TargetSecret *string `json:"targetSecret,omitempty"`
	// PrimaryKey +optional
	PrimaryKey *string `json:"primaryKey,omitempty"`
}

// StorageAccountStatus defines the observed state of StorageAccount
type StorageAccountStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=storageaccounts,shortName=storage,categories=all

// StorageAccount is the Schema for the publicips API
type StorageAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageAccountSpec   `json:"spec,omitempty"`
	Status StorageAccountStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StorageAccountList contains a list of StorageAccount
type StorageAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageAccount{}, &StorageAccountList{})
}
