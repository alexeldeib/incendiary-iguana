/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceBusNamespaceSpec defines the desired state of ServiceBusNamespace
type ServiceBusNamespaceSpec struct {
	// Name is the name of the security group.
	Name string `json:"name"`
	// Location osecurity group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup containsecurity group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// SKU is either basic or standard, representing the SKU of the IP in Azure.
	SKU ServiceBusNamespaceSku `json:"sku"`
	// TargetSecret +optional
	TargetSecret *string `json:"targetSecret,omitempty"`
	// PrimaryKey +optional
	PrimaryKey *string `json:"primaryKey,omitempty"`
	// SecondaryKey +optional
	SecondaryKey *string `json:"secondaryKey,omitempty"`
	// PrimaryConnectionString +optional
	PrimaryConnectionString *string `json:"primaryConnectionString,omitempty"`
	// SecondaryConnectionString +optional
	SecondaryConnectionString *string `json:"secondaryConnectionString,omitempty"`
}

type ServiceBusNamespaceSku struct {
	// Name of sku. Required for account creation; optional for update. Possible values include: 'Basic', 'Standard', 'Premium'
	Name SkuName `json:"name"`
	// Tier of corresponding SKU. Possible values include: 'C' (basic/standard), P (premium)
	Tier SkuName `json:"tier"`
	// Capacity of the cache to deploy. Valid values: for C (Basic/Standard) family (0, 1, 2, 3, 4, 5, 6), for P (Premium) family (1, 2, 3, 4).
	Capacity int32 `json:"capacity"`
}

// ServiceBusNamespaceStatus defines the observed state of ServiceBusNamespace
type ServiceBusNamespaceStatus struct {
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
// +kubebuilder:resource:path=servicebus,shortName=sb,categories=all

// ServiceBusNamespace is the Schema for the publicips API
type ServiceBusNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceBusNamespaceSpec   `json:"spec,omitempty"`
	Status ServiceBusNamespaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceBusNamespaceList contains a list of ServiceBusNamespace
type ServiceBusNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBusNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceBusNamespace{}, &ServiceBusNamespaceList{})
}
