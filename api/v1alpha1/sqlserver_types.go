/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SQLServerSpec defines the desired state of the SQL server.
type SQLServerSpec struct {
	// Name is the name of the resource.
	Name string `json:"name"`
	// Location is the region of the resource (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup is the resourceg group containing the resource.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// AllowAzureServiceAccess will allow access to this server from other Azure managed services if true.
	AllowAzureServiceAccess *bool `json:"allowAzureServiceAccess,omitempty"`
}

// SQLServerStatus defines the observed state of SQLServer
type SQLServerStatus struct {
	// State sync the status of the resource from Azure.
	State *string `json:"state,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sqlservers,shortName={sqlserver},categories=all

// SQLServer is the Schema for the SQL server API
type SQLServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SQLServerSpec   `json:"spec,omitempty"`
	Status SQLServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SQLServerList contains a list of SQLServers
type SQLServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQLServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SQLServer{}, &SQLServerList{})
}
