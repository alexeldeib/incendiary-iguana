/*
Copyright 2019 Alexander Eldeib.
*/

package loadbalancers

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

type Spec struct {
	internal *network.LoadBalancer
}

func NewSpec() *Spec {
	return &Spec{
		internal: &network.LoadBalancer{
			Sku: &network.LoadBalancerSku{
				Name: network.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{},
		},
	}
}

func NewSpecWithRemote(remote *network.LoadBalancer) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() network.LoadBalancer {
	return *s.internal
}

func Name(name string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Name = &name
	}
}

func Location(location string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Location = &location
	}
}

func Frontends(frontends []string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.FrontendIPConfigurations = &[]network.FrontendIPConfiguration{}

		for _, frontend := range frontends {
			// TODO(ace): name these properly? but remember they must be unique (maybe a map[string]string of name:ID?)
			parts := strings.Split(frontend, "/")
			resourceName := parts[len(parts)-1]

			item := network.FrontendIPConfiguration{
				Name: &resourceName,
				FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &network.PublicIPAddress{
						ID: &frontend,
					},
				},
			}

			*s.internal.FrontendIPConfigurations = append(*s.internal.FrontendIPConfigurations, item)
		}
	}
}

func Backends(backends []string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.BackendAddressPools = &[]network.BackendAddressPool{}
		for _, backend := range backends {
			*s.internal.BackendAddressPools = append(*s.internal.BackendAddressPools, network.BackendAddressPool{Name: &backend})
		}
	}
}

func Probes(ports []int) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Probes = &[]network.Probe{}
		for _, port := range ports {
			probe := network.Probe{
				Name: to.StringPtr(fmt.Sprintf("probe_%d", port)),
				ProbePropertiesFormat: &network.ProbePropertiesFormat{
					Protocol:          network.ProbeProtocolHTTPS,
					Port:              to.Int32Ptr(int32(port)),
					IntervalInSeconds: to.Int32Ptr(5),
					NumberOfProbes:    to.Int32Ptr(2),
					RequestPath:       to.StringPtr("/healthz"),
				},
			}
			*s.internal.Probes = append(*s.internal.Probes, probe)
		}
	}
}

func Rules() func(s *Spec) {
	return func(s *Spec) {
		// s.internal.LoadBalancingRules = &[]network.LoadBalancingRule{}
	}
}
