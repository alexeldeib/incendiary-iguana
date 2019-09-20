/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceBusKeySpec defines the desired state of ServiceBusKey
type ServiceBusKeySpec struct {
	// Name is the name of some resource in Azure.
	Name string `json:"name"`
	// ResourceGroup is the name of an Azure resource group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// TargetSecret is the name of the destination Kubernetes secret
	TargetSecret string `json:"targetSecret"`
	// PrimaryKey +optional
	PrimaryKey *string `json:"primaryKey,omitempty"`
	// SecondaryKey +optional
	SecondaryKey *string `json:"secondaryKey,omitempty"`
	// PrimaryConnectionString +optional
	PrimaryConnectionString *string `json:"primaryConnectionString,omitempty"`
	// SecondaryConnectionString +optional
	SecondaryConnectionString *string `json:"secondaryConnectionString,omitempty"`
}

// ServiceBusKeyStatus defines the observed state of ServiceBusKey
type ServiceBusKeyStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState *string `json:"provisioningState,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=servicebuskeys,shortName=sbkey,categories=all

// ServiceBusKey is the Schema for the publicips API
type ServiceBusKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceBusKeySpec   `json:"spec,omitempty"`
	Status ServiceBusKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceBusKeyList contains a list of ServiceBusKey
type ServiceBusKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBusKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceBusKey{}, &ServiceBusKeyList{})
}
