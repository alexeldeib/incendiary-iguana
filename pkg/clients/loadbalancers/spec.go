/*
Copyright 2019 Alexander Eldeib.
*/

package loadbalancers

// import (
// 	"fmt"
// 	"strings"

// 	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
// 	"github.com/Azure/go-autorest/autorest/to"
// 	"github.com/google/go-cmp/cmp"
// 	"github.com/sanity-io/litter"

// 	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
// 	"github.com/alexeldeib/incendiary-iguana/pkg/services/clientutil"
// )

// type Spec struct {
// 	internal *network.LoadBalancer
// }

// func NewSpec() *Spec {
// 	return &Spec{
// 		internal: &network.LoadBalancer{
// 			Sku: &network.LoadBalancerSku{
// 				Name: network.LoadBalancerSkuNameStandard,
// 			},
// 			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{},
// 		},
// 	}
// }

// func NewSpecWithRemote(remote *network.LoadBalancer) *Spec {
// 	return &Spec{
// 		internal: remote,
// 	}
// }

// func (s *Spec) Set(opts ...func(*Spec)) {
// 	for _, opt := range opts {
// 		opt(s)
// 	}
// }

// func (s *Spec) Build() network.LoadBalancer {
// 	return *s.internal
// }

// func Name(name string) func(s *Spec) {
// 	return func(s *Spec) {
// 		s.internal.Name = &name
// 	}
// }

// func Location(location string) func(s *Spec) {
// 	return func(s *Spec) {
// 		s.internal.Location = &location
// 	}
// }

// func Frontends(frontends []string) func(s *Spec) {
// 	return func(s *Spec) {
// 		s.internal.FrontendIPConfigurations = &[]network.FrontendIPConfiguration{}

// 		for _, frontend := range frontends {
// 			// TODO(ace): name these properly? but remember they must be unique (maybe a map[string]string of name:ID?)
// 			parts := strings.Split(frontend, "/")
// 			resourceName := parts[len(parts)-1]

// 			item := network.FrontendIPConfiguration{
// 				Name: &resourceName,
// 				FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
// 					PublicIPAddress: &network.PublicIPAddress{
// 						ID: &frontend,
// 					},
// 				},
// 			}

// 			*s.internal.FrontendIPConfigurations = append(*s.internal.FrontendIPConfigurations, item)
// 		}
// 	}
// }

// func Backends(backends []string) func(s *Spec) {
// 	return func(s *Spec) {
// 		s.internal.BackendAddressPools = &[]network.BackendAddressPool{}
// 		for _, backend := range backends {
// 			*s.internal.BackendAddressPools = append(*s.internal.BackendAddressPools, network.BackendAddressPool{Name: &backend})
// 		}
// 	}
// }

// func Probes(ports *[]int) func(s *Spec) {
// 	return func(s *Spec) {
// 		s.internal.Probes = &[]network.Probe{}
// 		if ports != nil {
// 			for _, port := range *ports {
// 				probe := network.Probe{
// 					Name: to.StringPtr(fmt.Sprintf("probe_%d", port)),
// 					ProbePropertiesFormat: &network.ProbePropertiesFormat{
// 						Protocol:          network.ProbeProtocolHTTPS,
// 						Port:              to.Int32Ptr(int32(port)),
// 						IntervalInSeconds: to.Int32Ptr(5),
// 						NumberOfProbes:    to.Int32Ptr(2),
// 						RequestPath:       to.StringPtr("/healthz"),
// 					},
// 				}
// 				*s.internal.Probes = append(*s.internal.Probes, probe)
// 			}
// 		}
// 	}
// }

// func Rules(rules *[]azurev1alpha1.RuleSpec) func(s *Spec) {
// 	return func(s *Spec) {
// 		s.internal.LoadBalancingRules = &[]network.LoadBalancingRule{}
// 		if rules != nil {
// 			for _, rule := range *rules {
// 				item := network.LoadBalancingRule{
// 					Name: &rule.Name,
// 					LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
// 						Protocol:                network.TransportProtocol(rule.Protocol),
// 						FrontendPort:            to.Int32Ptr(rule.FrontendPort),
// 						BackendPort:             to.Int32Ptr(rule.BackendPort),
// 						IdleTimeoutInMinutes:    to.Int32Ptr(4),
// 						EnableFloatingIP:        to.BoolPtr(false),
// 						EnableTCPReset:          to.BoolPtr(true),
// 						LoadDistribution:        network.LoadDistributionDefault,
// 						FrontendIPConfiguration: &network.SubResource{ID: &rule.Frontend},
// 						BackendAddressPool:      &network.SubResource{ID: &rule.BackendPool},
// 						Probe:                   &network.SubResource{ID: &rule.Probe},
// 					},
// 				}

// 				*s.internal.LoadBalancingRules = append(*s.internal.LoadBalancingRules, item)
// 			}
// 		}
// 	}
// }

// func (s *Spec) NeedsUpdate(local *azurev1alpha1.LoadBalancer) bool {
// 	return clientutil.Any([]func() bool{
// 		func() bool { return s.Name() == nil || local.Spec.Name != *s.Name() },
// 		func() bool { return s.Location() == nil || local.Spec.Location != *s.Location() },
// 		func() bool {
// 			val := local.Spec.Rules != nil && s.Rules() != nil && !cmp.Equal(*local.Spec.Rules, *s.Rules())
// 			fmt.Printf("val is: %v\n", val)
// 			litter.Dump(*local.Spec.Rules)
// 			litter.Dump(*s.Rules())
// 			litter.Dump(cmp.Diff(*local.Spec.Rules, *s.Rules()))
// 			return val
// 		},
// 		// func() bool { return Subnets(s) == nil || !cmp.Equal(local.Spec.Subnets, *Subnets(s)) },
// 	})
// }

// func (s *Spec) Name() *string {
// 	return s.internal.Name
// }

// func (s *Spec) Location() *string {
// 	return s.internal.Location
// }

// func (s *Spec) Rules() *[]azurev1alpha1.RuleSpec {
// 	if s.internal.LoadBalancerPropertiesFormat == nil || s.internal.LoadBalancerPropertiesFormat.LoadBalancingRules == nil {
// 		return nil
// 	}
// 	rules := &[]azurev1alpha1.RuleSpec{}
// 	for _, azureRule := range *s.internal.LoadBalancerPropertiesFormat.LoadBalancingRules {
// 		newRule := azurev1alpha1.RuleSpec{
// 			Name:         *azureRule.Name,
// 			Frontend:     *azureRule.LoadBalancingRulePropertiesFormat.FrontendIPConfiguration.ID,
// 			BackendPool:  *azureRule.LoadBalancingRulePropertiesFormat.BackendAddressPool.ID,
// 			Probe:        *azureRule.LoadBalancingRulePropertiesFormat.Probe.ID,
// 			Protocol:     string(azureRule.LoadBalancingRulePropertiesFormat.Protocol),
// 			FrontendPort: *azureRule.LoadBalancingRulePropertiesFormat.FrontendPort,
// 			BackendPort:  *azureRule.LoadBalancingRulePropertiesFormat.BackendPort,
// 		}
// 		*rules = append(*rules, newRule)
// 	}
// 	return rules
// }
