/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisSpec defines the desired state of Redis
type RedisSpec struct {
	// Name is the name of the security group.
	Name string `json:"name"`
	// Location osecurity group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup containsecurity group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID   string   `json:"subscriptionId"`
	Sku              RedisSku `json:"sku"`
	EnableNonSslPort bool     `json:"enableNonSslPort"`
}

type RedisSku struct {
	// Name of sku. Required for account creation; optional for update. Possible values include: 'Basic', 'Standard', 'Premium'
	Name SkuName `json:"name"`
	// Family of corresponding SKU. Possible values include: 'C' (basic/standard), P (premium)
	Family RedisSkuFamily `json:"family"`
	// Capacity of the cache to deploy. Valid values: for C (Basic/Standard) family (0, 1, 2, 3, 4, 5, 6), for P (Premium) family (1, 2, 3, 4).
	Capacity int32 `json:"capacity"`
}

type SkuName string

const (
	Basic    SkuName = "Basic"
	Premium  SkuName = "Premium"
	Standard SkuName = "Standard"
)

type RedisSkuFamily string

const (
	C RedisSkuFamily = "C"
	P RedisSkuFamily = "P"
)

// RedisStatus defines the observed state of Redis
type RedisStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState string `json:"provisioningState,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
	// ObservedGeneration is the iteration of user-provided spec which has already been reconciled.
	// This is used to decide when to re-reconcile changes.
	ObservedGeneration int64 `json:"observedGeneration"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=redis,categories=all

// Redis is the Schema for the redis API
type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSpec   `json:"spec,omitempty"`
	Status RedisStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisList contains a list of Redis
type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redis{}, &RedisList{})
}
