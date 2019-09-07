/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityGroupSpec defines the desired state of SecurityGroup
type SecurityGroupSpec struct {
	// Name is the name of the security group.
	Name string `json:"name"`
	// Location osecurity group (e.g., eastus2)
	Location string `json:"location"`
	// ResourceGroup containsecurity group.
	ResourceGroup string `json:"resourceGroup"`
	// SubscriptionID contains the Resource group. Is a GUID.
	SubscriptionID string         `json:"subscriptionId"`
	Rules          []SecurityRule `json:"rules,omitempty"`
}

// SecurityRule defines an allow/deny traffic rule on the security group.
type SecurityRule struct {
	// Name is the name of the rule.
	Name string `json:"name"`
	// Protocol - Network protocol this rule applies to. Possible values include: 'SecurityRuleProtocolTCP', 'SecurityRuleProtocolUDP', 'SecurityRuleProtocolIcmp', 'SecurityRuleProtocolEsp', 'SecurityRuleProtocolAsterisk'
	Protocol network.SecurityRuleProtocol `json:"protocol,omitempty"`
	// SourcePortRange - The source port or range. Integer or range between 0 and 65535. Asterisk '*' can also be used to match all ports.
	SourcePortRange *string `json:"sourcePortRange"`
	// DestinationPortRange - The destination port or range. Integer or range between 0 and 65535. Asterisk '*' can also be used to match all ports.
	DestinationPortRange *string `json:"destinationPortRange"`
	// SourceAddressPrefix - The CIDR or source IP range. Asterisk '*' can also be used to match all source IPs. Default tags such as 'VirtualNetwork', 'AzureLoadBalancer' and 'Internet' can also be used. If this is an ingress rule, specifies where network traffic originates from.
	SourceAddressPrefix *string `json:"sourceAddressPrefix"`
	// DestinationAddressPrefix - The destination address prefix. CIDR or destination IP range. Asterisk '*' can also be used to match all source IPs. Default tags such as 'VirtualNetwork', 'AzureLoadBalancer' and 'Internet' can also be used.
	DestinationAddressPrefix *string `json:"destinationAddressPrefix,omitempty"`
	// Access - The network traffic is allowed or denied. Possible values include: 'SecurityRuleAccessAllow', 'SecurityRuleAccessDeny'
	Access network.SecurityRuleAccess `json:"access"`
	// Priority - The priority of the rule. The value can be between 100 and 4096. The priority number must be unique for each rule in the collection. The lower the priority number, the higher the priority of the rule.
	Priority *int32 `json:"priority"`
	// Direction - The direction of the rule. The direction specifies if rule will be evaluated on incoming or outgoing traffic. Possible values include: 'SecurityRuleDirectionInbound', 'SecurityRuleDirectionOutbound'
	Direction network.SecurityRuleDirection `json:"direction"`
}

// SecurityGroupStatus defines the observed state of SecurityGroup
type SecurityGroupStatus struct {
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
// +kubebuilder:resource:path=securitygroups,shortName=sg,categories=all

// SecurityGroup is the Schema for the securitygroups API
type SecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityGroupSpec   `json:"spec,omitempty"`
	Status SecurityGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityGroupList contains a list of SecurityGroup
type SecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityGroup{}, &SecurityGroupList{})
}
