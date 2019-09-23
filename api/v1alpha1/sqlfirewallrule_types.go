/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SQLFirewallRuleSpec defines the desired state of the firewall rule.
type SQLFirewallRuleSpec struct {
	// Name is the name of the resource.
	Name string `json:"name"`
	// Server is the name of the SQL server this rule should apply to.
	Server string `json:"server"`
	// ResourceGroup is the resourceg group containing the resource.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the resource group. Is a GUID.
	SubscriptionID string `json:"subscriptionId"`
	// Start is the beginning of the IP range to allow
	Start string `json:"start"`
	// End is the end of the IP range to allow
	End string `json:"end"`
}

// SQLFirewallRuleStatus defines the observed state of SQLFirewallRule
type SQLFirewallRuleStatus struct {
	// State sync the status of the resource from Azure.
	State *string `json:"state,omitempty"`
	// ID is the fully qualified Azure resource ID.
	ID *string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sqlfirewalls,shortName={sqlfw},categories=all

// SQLFirewallRule is the Schema for the SQL server API
type SQLFirewallRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SQLFirewallRuleSpec   `json:"spec,omitempty"`
	Status SQLFirewallRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SQLFirewallRuleList contains a list of SQLFirewallRules
type SQLFirewallRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQLFirewallRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SQLFirewallRule{}, &SQLFirewallRuleList{})
}
