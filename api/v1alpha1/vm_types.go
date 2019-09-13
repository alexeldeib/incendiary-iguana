/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VMSpec defines the desired state of VM
type VMSpec struct {
	// Name is the name of the security group.
	Name string `json:"name"`
	// Location osecurity group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup containsecurity group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// SKU is the sku of the machine in Azure, e.g. Standard_E4_v3
	SKU string `json:"sku"`
	// CustomData is the cloud-init/script user data for the machine.
	Customdata *string `json:"customData,omitempty"`
	// SSHPublicKey is the key of the of the provisioned user on the VM.
	SSHPublicKey *string `json:"sshPublicKey,omitempty"`
	// PrimaryNIC is the Azure ID of the primary NIC on this machine.
	PrimaryNIC string `json:"primaryNic"`
	// SecondaryNICs is the list of IDs of non-primary NICs on this machine. +optional
	SecondaryNICs *[]string `json:"secondaryNics,omitempty"`
	// DiskSize is the size of the OS disk in GB
	DiskSize int32 `json:"diskSize"`
}

// VMStatus defines the observed state of VM
type VMStatus struct {
	// ProvisioningState sync the provisioning status of the resource from Azure.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=vms,shortName=vm,categories=all

// VM is the Schema for the VMs API
type VM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMSpec   `json:"spec,omitempty"`
	Status VMStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VMList contains a list of VM
type VMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VM `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VM{}, &VMList{})
}
