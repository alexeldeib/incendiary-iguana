/*
Copyright 2019 Alexander Eldeib.
*/

package resourcegroups

import (
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/clientutil"
)

type Spec struct {
	internal *resources.Group
}

func NewSpec() *Spec {
	return &Spec{
		internal: &resources.Group{},
	}
}

func NewSpecWithRemote(remote *resources.Group) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) Build() resources.Group {
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

func (s *Spec) NeedsUpdate(local *azurev1alpha1.ResourceGroup) bool {
	return clientutil.Any([]func() bool{
		func() bool { return s.Name() == nil || local.Spec.Name != *s.Name() },
		func() bool { return s.Location() == nil || local.Spec.Location != *s.Location() },
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
	if s.internal.Properties == nil {
		return nil
	}
	return s.internal.Properties.ProvisioningState
}
