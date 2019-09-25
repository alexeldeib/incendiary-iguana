/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageKeySpec defines the desired state of StorageKey
type StorageKeySpec struct {
	// Name is the name of the resource.
	Name string `json:"name"`
	// ResourceGroup containing the resource.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// TargetSecret +optional
	TargetSecret *string `json:"targetSecret,omitempty"`
	// PrimaryKey +optional
	PrimaryKey *string `json:"primaryKey,omitempty"`
}

// StorageKeyStatus defines the observed state of StorageKey
type StorageKeyStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=storagekeys,categories=all

// StorageKey is the Schema for the publicips API
type StorageKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageKeySpec   `json:"spec,omitempty"`
	Status StorageKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StorageKeyList contains a list of StorageKey
type StorageKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageKey{}, &StorageKeyList{})
}
