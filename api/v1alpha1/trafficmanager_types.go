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
	Name                 string          `json:"name"`
	SubscriptionID       string          `json:"subscriptionID"`
	ResourceGroup        string          `json:"resourceGroup"`
	ProfileStatus        string          `json:"profileStatus"`
	TrafficRoutingMethod string          `json:"trafficRoutingMethod"`
	Endpoints            *[]EndpointSpec `json:"endpoints,omitEmpty"`
	DNSConfig            DNSConfig       `json:"dnsConfig,omitempty"`
	MonitorConfig        MonitorConfig   `json:"monitorConfig,omitempty"`
}

type DNSConfig struct {
	RelativeName *string `json:"relativeName"`
	TTL          *int64  `json:"ttl,omitempty"`
}

type MonitorConfig struct {
	// Protocol - The protocol (HTTP, HTTPS or TCP) used to probe for endpoint health. Possible values include: 'HTTP', 'HTTPS', 'TCP'
	Protocol string `json:"protocol,omitempty"`
	// Port - The TCP port used to probe for endpoint health.
	Port *int64 `json:"port,omitempty"`
	// Path - The path relative to the endpoint domain name used to probe for endpoint health.
	Path *string `json:"path,omitempty"`
	// IntervalInSeconds - The monitor interval for endpoints in this profile. This is the interval at which Traffic Manager will check the health of each endpoint in this profile.
	IntervalInSeconds *int64 `json:"intervalInSeconds,omitempty"`
	// TimeoutInSeconds - The monitor timeout for endpoints in this profile. This is the time that Traffic Manager allows endpoints in this profile to response to the health check.
	TimeoutInSeconds *int64 `json:"timeoutInSeconds,omitempty"`
	// ToleratedNumberOfFailures - The number of consecutive failed health check that Traffic Manager tolerates before declaring an endpoint in this profile Degraded after the next failed health check.
	ToleratedNumberOfFailures *int64 `json:"toleratedNumberOfFailures,omitempty"`
	// CustomHeaders - List of custom headers.
	CustomHeaders *[]MonitorConfigCustomHeadersItem `json:"customHeaders,omitempty"`
	// ExpectedStatusCodeRanges - List of expected status code ranges.
	ExpectedStatusCodeRanges *[]MonitorConfigExpectedStatusCodeRangesItem `json:"expectedStatusCodeRanges,omitempty"`
}

type MonitorConfigCustomHeadersItem struct {
	// Name - Header name.
	Name *string `json:"name,omitempty"`
	// Value - Header value.
	Value *string `json:"value,omitempty"`
}

type MonitorConfigExpectedStatusCodeRangesItem struct {
	// Min - Min status code.
	Min *int32 `json:"min,omitempty"`
	// Max - Max status code.
	Max *int32 `json:"max,omitempty"`
}

type EndpointProperties struct {
	// Target - The fully-qualified DNS name or IP address of the endpoint. Traffic Manager returns this value in DNS responses to direct traffic to this endpoint.
	Target *string `json:"target,omitempty"`
	// Weight - The weight of this endpoint when using the 'Weighted' traffic routing method. Possible values are from 1 to 1000.
	Weight *int64 `json:"weight,omitempty"`
	// Priority - The priority of this endpoint when using the 'Priority' traffic routing method. Possible values are from 1 to 1000, lower values represent higher priority. This is an optional parameter.  If specified, it must be specified on all endpoints, and no two endpoints can share the same priority value.
	Priority int64 `json:"priority,omitempty"`
	// EndpointLocation - Specifies the location of the external or nested endpoints when using the 'Performance' traffic routing method.
	EndpointLocation string `json:"endpointLocation,omitempty"`
	// CustomHeaders - List of custom headers.
	CustomHeaders *[]MonitorConfigCustomHeadersItem `json:"customHeaders,omitempty"`
}

type EndpointSpec struct {
	Name       string             `json:"name"`
	Properties EndpointProperties `json:"properties"`
}

// +kubebuilder:object:generate=true

type EndpointStatus struct {
	MonitorStatus string `json:"monitorStatus,omitempty"`
}

// TrafficManagerStatus defines the observed state of TrafficManager
type TrafficManagerStatus struct {
	// ID is the fully qualified Azure resource ID.
	ID                   *string           `json:"id"`
	FQDN                 *string           `json:"fqdn,omitempty"`
	ProfileStatus        string            `json:"profileStatus"`
	ProfileMonitorStatus string            `json:"profileMonitorStatus"`
	EndpointStatus       *[]EndpointStatus `json:"endpointStatus,omitempty"`
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
