/*
Copyright 2019 Alexander Eldeib.
*/

package subnets

import (
	"github.com/google/go-cmp/cmp"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/clientutil"
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

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() network.Subnet {
	return *s.internal
}

func Name(name string) func(s *Spec) {
	return func(s *Spec) {
		s.internal.Name = &name
	}
}

func Address(cidr string) func(s *Spec) {
	return func(s *Spec) {
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
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.Subnet) bool {
	return clientutil.Any([]func() bool{
		func() bool { return s.Name() == nil || local.Spec.Name != *s.Name() },
		func() bool { return s.Address() == nil || !cmp.Equal(local.Spec.Subnet, *s.Address()) },
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

func (s *Spec) Name() *string {
	return s.internal.Name
}

func (s *Spec) ID() *string {
	return s.internal.ID
}

func (s *Spec) Address() *string {
	if s.internal.SubnetPropertiesFormat == nil {
		return nil
	}
	return s.internal.SubnetPropertiesFormat.AddressPrefix
}

func (s *Spec) State() *string {
	if s.internal.SubnetPropertiesFormat == nil {
		return nil
	}
	return s.internal.SubnetPropertiesFormat.ProvisioningState
}
