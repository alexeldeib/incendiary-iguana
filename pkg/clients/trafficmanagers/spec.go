/*
Copyright 2019 Alexander Eldeib.
*/

package trafficmanagers

import (
	"github.com/Azure/azure-sdk-for-go/services/trafficmanager/mgmt/2018-04-01/trafficmanager"
	"github.com/Azure/go-autorest/autorest/to"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type Spec struct {
	internal *trafficmanager.Profile
}

func NewSpec() *Spec {
	return &Spec{
		internal: &trafficmanager.Profile{
			ProfileProperties: &trafficmanager.ProfileProperties{
				ProfileStatus:        trafficmanager.ProfileStatusEnabled,
				TrafficRoutingMethod: trafficmanager.Weighted,
				MonitorConfig: &trafficmanager.MonitorConfig{
					Protocol:                  trafficmanager.MonitorProtocol("HTTPS"),
					Port:                      to.Int64Ptr(443),
					Path:                      to.StringPtr("/healthz"),
					IntervalInSeconds:         to.Int64Ptr(10),
					TimeoutInSeconds:          to.Int64Ptr(5),
					ToleratedNumberOfFailures: to.Int64Ptr(3),
					CustomHeaders:             &[]trafficmanager.MonitorConfigCustomHeadersItem{},
					ExpectedStatusCodeRanges:  &[]trafficmanager.MonitorConfigExpectedStatusCodeRangesItem{},
				},
				Endpoints: &[]trafficmanager.Endpoint{},
				DNSConfig: &trafficmanager.DNSConfig{},
			},
			Location: to.StringPtr("global"),
		},
	}
}

func NewSpecWithRemote(remote *trafficmanager.Profile) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Build() trafficmanager.Profile {
	return *s.internal
}

func (s *Spec) Name(name *string) {
	s.internal.Name = name
}

func (s *Spec) Health(path *string) {
	s.internal.MonitorConfig.Path = path
}

func (s *Spec) Location(location *string) {
	s.internal.Location = location
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.TrafficManager) bool {
	return any([]func() bool{
		func() bool { return Name(s) == nil || local.Spec.Name != *Name(s) },
		func() bool {
			return Status(s) != nil && *Status(s) != trafficmanager.ProfileStatus(local.Spec.ProfileStatus)
		},
		func() bool { return Name(s) == nil || local.Spec.Name != *Name(s) },
	})
}

func Name(s *Spec) *string {
	return s.internal.Name
}

func ID(s *Spec) *string {
	return s.internal.ID
}

func Status(s *Spec) *trafficmanager.ProfileStatus {
	if s.internal.ProfileProperties == nil {
		return nil
	}
	return &s.internal.ProfileProperties.ProfileStatus
}

func any(funcs []func() bool) bool {
	for _, f := range funcs {
		if f() {
			return true
		}
	}
	return false
}
