/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisKeySpec defines the desired state of RedisKey
type RedisKeySpec struct {
	// Name is the name of some resource in Azure.
	Name string `json:"name"`
	// ResourceGroup is the name of an Azure resource group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// TargetSecret
	TargetSecret string `json:"targetSecret"`
	// PrimaryKey +optional
	PrimaryKey *string `json:"primaryKey,omitempty"`
	// SecondaryKey +optional
	SecondaryKey *string `json:"secondaryKey,omitempty"`
}

// RedisKeyStatus defines the observed state of RedisKey
type RedisKeyStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState *string `json:"provisioningState,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rediskeys,shortName=rediskey,categories=all

// RedisKey is the Schema for the publicips API
type RedisKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisKeySpec   `json:"spec,omitempty"`
	Status RedisKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RedisKeyList contains a list of RedisKey
type RedisKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisKey{}, &RedisKeyList{})
}
