/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkInterfaceSpec defines the desired state of NetworkInterface
type NetworkInterfaceSpec struct {
	// Name is the name of the security group.
	Name string `json:"name"`
	// Location osecurity group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup containsecurity group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// Network is the name of the VNet containing this subnet
	Network string `json:"network"`
	// Subnet contains the name to an Azure subnet which this NIC should belong to.
	Subnet string `json:"subnet"`
	// IPConfigurations is an array of IP configurations belonging to this interface.
	IPConfigurations *[]InterfaceIPConfig `json:"ipConfigurations,omitempty"`
}

// InterfaceIPConfig describes a single IP configuration for a NIC.
type InterfaceIPConfig struct {
	// PublicIP contains an optional reference to an existing IP address to bind to this NIC.
	PublicIP *ResourceReference `json:"publicIP,omitempty"`
	// PrivateIP contains an optional private IP address to bind to this NIC.
	PrivateIP *string `json:"privateIP,omitempty"`
	// BackendPoolReferences contains an optional reference to a Load Balancer backend pool for this configuration.
	LoadBalancers *[]BackendPoolReference `json:"loadBalancers,omitempty"`
}

// ResourceReference contains information to identify a generic Azure resource.
type ResourceReference struct {
	// Name is the name of the referenced resource.
	Name string `json:"name"`
	// ResourceGroup contain the referenced resource.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
}

// BackendPoolReference contains a reference to a Load Balancer backend pool for this configuration.
type BackendPoolReference struct {
	// Name is the name of the referenced resource.
	Name string `json:"name"`
	// LoadBalancer is the name of the associated Load balancer.
	LoadBalancer string `json:"loadBalancer"`
	// ResourceGroup contain the referenced resource.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
}

// NetworkInterfaceStatus defines the observed state of NetworkInterface
type NetworkInterfaceStatus struct {
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
// +kubebuilder:resource:path=netwworkinterfaces,shortName={nic,nics},categories=all

// NetworkInterface is the Schema for the networkinterfaces API
type NetworkInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkInterfaceSpec   `json:"spec,omitempty"`
	Status NetworkInterfaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NetworkInterfaceList contains a list of NetworkInterface
type NetworkInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkInterface{}, &NetworkInterfaceList{})
}
