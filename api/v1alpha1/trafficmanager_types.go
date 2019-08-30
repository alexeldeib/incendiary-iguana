/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TrafficManagerSpec defines the desired state of TrafficManager
type TrafficManagerSpec struct {
	Name                 string `json:"name"`
	SubscriptionID       string `json:"subscriptionID"`
	ResourceGroup        string `json:"resourceGroup"`
	ProfileStatus        string `json:"profileStatus"`
	TrafficRoutingMethod string `json:"trafficRoutingMethod"`
	DNSName              string `json:"dnsName"`
	Protocol             string `json:"protocol"`
	Healthcheck          string `json:"healthcheck"`
	IntervalInSeconds    *int   `json:"intervalInSeconds,omitempty"` // +optional
	TimeoutInSeconds     *int   `json:"timeoutInSeconds,omitempty"`  // +optional
}

// TrafficManagerStatus defines the observed state of TrafficManager
type TrafficManagerStatus struct {
	// ID is the fully qualified Azure resource ID.
	ID                   string  `json:"id"`
	FQDN                 *string `json:"fqdn,omitempty"`
	ProfileStatus        string  `json:"profileStatus"`
	ProfileMonitorStatus string  `json:"profileMonitorStatus"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=trafficmanagers,categories=all,shortName=tm

// TrafficManager is the Schema for the trafficmanagers API
type TrafficManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrafficManagerSpec   `json:"spec,omitempty"`
	Status TrafficManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TrafficManagerList contains a list of TrafficManager
type TrafficManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrafficManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrafficManager{}, &TrafficManagerList{})
}
