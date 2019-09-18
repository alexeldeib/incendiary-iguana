/*
Copyright 2019 Alexander Eldeib.
*/

package publicips

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/clientutil"
)

type Spec struct {
	internal *network.PublicIPAddress
}

func NewSpec() *Spec {
	return &Spec{
		internal: &network.PublicIPAddress{},
	}
}

func NewSpecWithRemote(remote *network.PublicIPAddress) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() network.PublicIPAddress {
	return *s.internal
}

func Name(name *string) func(*Spec) {
	return func(s *Spec) {
		s.internal.Name = name
	}
}

func Location(location *string) func(*Spec) {
	return func(s *Spec) {
		s.internal.Location = location
	}
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.PublicIP) bool {
	return clientutil.Any([]func() bool{
		clientutil.StringPtrChanged(s.Name(), &local.Spec.Name),
		clientutil.StringPtrChanged(s.Location(), &local.Spec.Location),
		clientutil.StringPtrChanged(s.SKU(), local.Spec.SKU),
		// clientutil.StringPtrChanged(s.AllocationMethod(), local.Spec.AllocationMethod),
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

func (s *Spec) State() *string {
	if s.internal.PublicIPAddressPropertiesFormat == nil {
		return nil
	}
	return s.internal.PublicIPAddressPropertiesFormat.ProvisioningState
}

func (s *Spec) SKU() *string {
	if s.internal.Sku == nil {
		return nil
	}
	return to.StringPtr(string(s.internal.Sku.Name))
}

func (s *Spec) AllocationMethod() *string {
	if s.internal.PublicIPAddressPropertiesFormat == nil {
		return nil
	}
	return to.StringPtr(string(s.internal.PublicIPAddressPropertiesFormat.PublicIPAllocationMethod))
}
