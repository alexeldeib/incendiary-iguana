/*
Copyright 2019 Alexander Eldeib.
*/

package resourcegroups

import (
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
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

func (s *Spec) Build() resources.Group {
	return *s.internal
}

func (s *Spec) Name(name *string) {
	s.internal.Name = name
}

func (s *Spec) Location(location *string) {
	s.internal.Location = location
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.ResourceGroup) bool {
	return any([]func() bool{
		func() bool { return Name(s) == nil || local.Spec.Name != *Name(s) },
		func() bool { return Location(s) == nil || local.Spec.Location != *Location(s) },
	})
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

func State(s *Spec) *string {
	if s.internal.Properties == nil {
		return nil
	}
	return s.internal.Properties.ProvisioningState
}

func any(funcs []func() bool) bool {
	for _, f := range funcs {
		if f() {
			return true
		}
	}
	return false
}
