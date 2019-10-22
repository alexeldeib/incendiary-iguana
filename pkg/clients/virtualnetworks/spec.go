/*
Copyright 2019 Alexander Eldeib.
*/

package virtualnetworks

import (
	"github.com/google/go-cmp/cmp"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/clientutil"
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

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() network.VirtualNetwork {
	return *s.internal
}

func Name(name *string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Name = name
	}
}

func Location(location *string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Location = location
	}
}

func AddressSpaces(cidrs []string) func(s *Spec) {
	return func(s *Spec) {
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
}

func AddressSpace(cidr string) func(s *Spec) {
	return func(s *Spec) {
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
}

func RemoveAddressSpace(cidr string) func(s *Spec) {
	return func(s *Spec) {
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
}

func Subnet(name, cidr string) func(s *Spec) {
	return func(s *Spec) {
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
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.VirtualNetwork) bool {
	return clientutil.Any([]func() bool{
		func() bool { return s.Name() == nil || local.Spec.Name != *s.Name() },
		func() bool { return s.Location() == nil || local.Spec.Location != *s.Location() },
		func() bool { return s.Addresses() == nil || !cmp.Equal(local.Spec.Addresses, *s.Addresses()) },
		// func() bool { return Subnets(s) == nil || !cmp.Equal(local.Spec.Subnets, *Subnets(s)) },
	})
}

func (s *Spec) Name() *string {
	return s.internal.Name
}

func (s *Spec) Location() *string {
	return s.internal.Location
}

func (s *Spec) ID() *string {
	return s.internal.ID
}

func (s *Spec) Addresses() *[]string {
	if s.internal.VirtualNetworkPropertiesFormat == nil || s.internal.VirtualNetworkPropertiesFormat.AddressSpace == nil {
		return nil
	}
	return s.internal.VirtualNetworkPropertiesFormat.AddressSpace.AddressPrefixes
}

func (s *Spec) Subnets() *[]string {
	if s.internal.VirtualNetworkPropertiesFormat == nil || s.internal.VirtualNetworkPropertiesFormat.Subnets == nil {
		return nil
	}
	subnets := &[]string{}
	for _, subnet := range *s.internal.VirtualNetworkPropertiesFormat.Subnets {
		*subnets = append(*subnets, *subnet.AddressPrefix)
	}
	return subnets
}

func (s *Spec) State() *string {
	if s.internal.VirtualNetworkPropertiesFormat == nil {
		return nil
	}
	return s.internal.VirtualNetworkPropertiesFormat.ProvisioningState
}
