/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoadBalancerSpec defines the desired state of Load Balancer
type LoadBalancerSpec struct {
	// Name is the name of the Azure LoadBalancer.
	Name string `json:"name"`
	// Location of the resource group (e.g., eastus2 or "West US")
	Location string `json:"location"`
	// ResourceGroup contains the LoadBalancer.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// SKU is either basic or standard.
	SKU *string `json:"sku,omitempty"`
	// FrontendIPConfigurations []FrontendIPConfigurationSpec `json:"frontendIPConfigurations"`
	// BackendAddressPools      []BackendAddressPoolSpec      `json:"backendAddressPools"`
	// OutboundRules            []OutboundRuleSpec            `json:"frontendIPConfigurations"`
}

// FrontendIPConfigurationSpec defines the front end ip configuration of LoadBalancer
type FrontendIPConfigurationSpec struct {
	Name     string `json:"name,omitempty"`
	Subnet   string `json:"subnet"`
	PublicIP string `json:"publicIP,omitempty"`
}

// LoadBalancerStatus defines the observed state of Load Balancer
type LoadBalancerStatus struct {
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
// +kubebuilder:resource:path=loadbalancers,shortName=lb,categories=all

// LoadBalancer is the Schema for the loadbalancers API
type LoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadBalancerSpec   `json:"spec,omitempty"`
	Status LoadBalancerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LoadBalancerList contains a list of Load Balancer
type LoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadBalancer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoadBalancer{}, &LoadBalancerList{})
}
