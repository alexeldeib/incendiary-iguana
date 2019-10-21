/*
Copyright 2019 Alexander Eldeib.
*/

package subnets

import (
	"github.com/google/go-cmp/cmp"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type Spec struct {
	internal *network.Subnet
}

func NewSpec() *Spec {
	return &Spec{
		internal: &network.Subnet{},
	}
}

func NewSpecWithRemote(remote *network.Subnet) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Build() network.Subnet {
	return *s.internal
}

func (s *Spec) Name(name string) {
	s.internal.Name = &name
}

func (s *Spec) Address(cidr string) {
	s.initialize(
		[]func() bool{
			func() bool { return s.internal.SubnetPropertiesFormat == nil },
		},
		[]func(){
			func() { s.internal.SubnetPropertiesFormat = &network.SubnetPropertiesFormat{} },
		},
	)
	s.internal.SubnetPropertiesFormat.AddressPrefix = &cidr
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.Subnet) bool {
	return any([]func() bool{
		func() bool { return Name(s) == nil || local.Spec.Name != *Name(s) },
		func() bool { return Address(s) == nil || !cmp.Equal(local.Spec.Subnet, *Address(s)) },
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

func ID(s *Spec) *string {
	return s.internal.ID
}

func Address(s *Spec) *string {
	if s.internal.SubnetPropertiesFormat == nil {
		return nil
	}
	return s.internal.SubnetPropertiesFormat.AddressPrefix
}

func State(s *Spec) *string {
	if s.internal.SubnetPropertiesFormat == nil {
		return nil
	}
	return s.internal.SubnetPropertiesFormat.ProvisioningState
}

func any(funcs []func() bool) bool {
	for _, f := range funcs {
		if f() {
			return true
		}
	}
	return false
}
