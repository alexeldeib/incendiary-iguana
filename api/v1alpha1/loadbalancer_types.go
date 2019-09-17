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
	// +kubebuilder:validation:MinItems=1
	// Frontends is a list of fully qualified resource IDs to Azure public IPs.
	Frontends []string `json:"frontends"`
	// +kubebuilder:validation:MinItems=1
	// BackendPools is a list names of backend pools to create for this Load Balancers.
	BackendPools []string `json:"backendPools"`
	// Rules is the list of load balancing rules.
	Rules *[]RuleSpec `json:"rules,omitempty"`
	// Probes is the list of load balancing health probes.
	Probes *[]int `json:"probes,omitempty"`
}

type RuleSpec struct {
	// Name is the name of the load balancing rule.
	Name string `json:"name"`
	// Frontend fully qualified reference to a frontend IP addresses.
	Frontend string `json:"frontendIPConfiguration"`
	// BackendPool - A reference to a pool of DIPs. Inbound traffic is randomly load balanced across IPs in the backend IPs.
	BackendPool string `json:"backendPool"`
	// Probe - The reference of the load balancer probe used by the load balancing rule.
	Probe string `json:"probe"`
	// Protocol is the transport protocol used by the load balancing rule. Possible values include: 'TransportProtocolUDP', 'TransportProtocolTCP', 'TransportProtocolAll'
	Protocol string `json:"protocol"`
	// FrontendPort is the port for the external endpoint. Port numbers for each rule must be unique within the Load Balancer. Acceptable values are between 0 and 65534. Note that value 0 enables "Any Port".
	FrontendPort int32 `json:"frontendPort"`
	// BackendPort - The port used for internal connections on the endpoint. Acceptable values are between 0 and 65535. Note that value 0 enables "Any Port".
	BackendPort int32 `json:"backendPort"`
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
