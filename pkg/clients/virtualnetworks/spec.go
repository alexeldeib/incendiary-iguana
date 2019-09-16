/*
Copyright 2019 Alexander Eldeib.
*/

package virtualnetworks

import (
	"github.com/google/go-cmp/cmp"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/clientutil"
)

type Spec struct {
	internal *network.VirtualNetwork
}

func NewSpec() *Spec {
	return &Spec{
		internal: &network.VirtualNetwork{},
	}
}

func NewSpecWithRemote(remote *network.VirtualNetwork) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Build() network.VirtualNetwork {
	return *s.internal
}

func (s *Spec) Name(name *string) {
	s.internal.Name = name
}

func (s *Spec) Location(location *string) {
	s.internal.Location = location
}

func (s *Spec) AddressSpaces(cidrs []string) {
	clientutil.Initialize(
		[]func() bool{
			func() bool { return s.internal.VirtualNetworkPropertiesFormat == nil },
			func() bool { return s.internal.VirtualNetworkPropertiesFormat.AddressSpace == nil },
			func() bool { return s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes == nil },
		},
		[]func(){
			func() { s.internal.VirtualNetworkPropertiesFormat = &network.VirtualNetworkPropertiesFormat{} },
			func() { s.internal.VirtualNetworkPropertiesFormat.AddressSpace = &network.AddressSpace{} },
			func() {
				s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes = &[]string{}
			},
		},
	)
	for _, cidr := range cidrs {
		found := false
		for _, prefix := range *s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes {
			if prefix == cidr {
				found = true
			}
		}
		if !found {
			*s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes = append(
				*s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes,
				cidr,
			)
		}
	}
}

func (s *Spec) AddressSpace(cidr string) {
	clientutil.Initialize(
		[]func() bool{
			func() bool { return s.internal.VirtualNetworkPropertiesFormat == nil },
			func() bool { return s.internal.VirtualNetworkPropertiesFormat.AddressSpace == nil },
			func() bool { return s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes == nil },
		},
		[]func(){
			func() { s.internal.VirtualNetworkPropertiesFormat = &network.VirtualNetworkPropertiesFormat{} },
			func() { s.internal.VirtualNetworkPropertiesFormat.AddressSpace = &network.AddressSpace{} },
			func() { s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes = &[]string{} },
		},
	)
	found := false
	for _, prefix := range *s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes {
		if prefix == cidr {
			found = true
		}
	}
	if !found {
		*s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes = append(
			*s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes,
			cidr,
		)
	}
}

func (s *Spec) RemoveAddressSpace(cidr string) {
	clientutil.Initialize(
		[]func() bool{
			func() bool { return s.internal.VirtualNetworkPropertiesFormat == nil },
			func() bool { return s.internal.VirtualNetworkPropertiesFormat.AddressSpace == nil },
			func() bool { return s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes == nil },
		},
		[]func(){
			func() { s.internal.VirtualNetworkPropertiesFormat = &network.VirtualNetworkPropertiesFormat{} },
			func() { s.internal.VirtualNetworkPropertiesFormat.AddressSpace = &network.AddressSpace{} },
			func() { s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes = &[]string{} },
		},
	)
	for i, prefix := range *s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes {
		if prefix == cidr {
			*s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes = append(
				(*s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes)[:i],
				(*s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes)[:i+1]...,
			)
		}
	}
}

func (s *Spec) Subnet(name, cidr string) {
	clientutil.Initialize(
		[]func() bool{
			func() bool { return s.internal.VirtualNetworkPropertiesFormat == nil },
			func() bool { return s.internal.VirtualNetworkPropertiesFormat.Subnets == nil },
		},
		[]func(){
			func() { s.internal.VirtualNetworkPropertiesFormat = &network.VirtualNetworkPropertiesFormat{} },
			func() { s.internal.VirtualNetworkPropertiesFormat.Subnets = &[]network.Subnet{} },
		},
	)

	found := false
	for _, subnet := range *s.internal.VirtualNetworkPropertiesFormat.Subnets {
		if *subnet.Name == name {
			subnet.AddressPrefix = &cidr
			found = true
		}
	}

	if !found {
		*s.internal.VirtualNetworkPropertiesFormat.Subnets = append(
			*s.internal.VirtualNetworkPropertiesFormat.Subnets,
			network.Subnet{
				Name: &name,
				SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
					AddressPrefix: &cidr,
				},
			},
		)
	}
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.VirtualNetwork) bool {
	return clientutil.Any([]func() bool{
		func() bool { return Name(s) == nil || local.Spec.Name != *Name(s) },
		func() bool { return Location(s) == nil || local.Spec.Location != *Location(s) },
		func() bool { return Addresses(s) == nil || !cmp.Equal(local.Spec.Addresses, *Addresses(s)) },
		// func() bool { return Subnets(s) == nil || !cmp.Equal(local.Spec.Subnets, *Subnets(s)) },
	})
}

func (s *Spec) initialize(detectors []func() bool, remediators []func()) {
	for idx, f := range detectors {
		if f() {
			remediators[idx]()
		}
	}
}

func Name(s *Spec) *string {
	return s.internal.Name
}

func Location(s *Spec) *string {
	return s.internal.Location
}

func ID(s *Spec) *string {
	return s.internal.ID
}

func Addresses(s *Spec) *[]string {
	if s.internal.VirtualNetworkPropertiesFormat == nil || s.internal.VirtualNetworkPropertiesFormat.AddressSpace == nil {
		return nil
	}
	return s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes
}

func Subnets(s *Spec) *[]string {
	if s.internal.VirtualNetworkPropertiesFormat == nil || s.internal.VirtualNetworkPropertiesFormat.Subnets == nil {
		return nil
	}
	subnets := &[]string{}
	for _, subnet := range *s.internal.VirtualNetworkPropertiesFormat.Subnets {
		*subnets = append(*subnets, *subnet.AddressPrefix)
	}
	return subnets
}

func State(s *Spec) *string {
	if s.internal.VirtualNetworkPropertiesFormat == nil {
		return nil
	}
	return s.internal.VirtualNetworkPropertiesFormat.ProvisioningState
}

func any(funcs []func() bool) bool {
	for _, f := range funcs {
		if f() {
			return true
		}
	}
	return false
}
