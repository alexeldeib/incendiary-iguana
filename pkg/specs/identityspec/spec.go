/*
Copyright 2019 Alexander Eldeib.
*/

package identityspec

import (
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/Azure/go-autorest/autorest/to"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type Spec struct {
	internal *msi.Identity
}

func New() *Spec {
	return &Spec{
		internal: &msi.Identity{},
	}
}

func NewFromExisting(remote *msi.Identity) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Build() msi.Identity {
	return *s.internal
}

func (s *Spec) Name(name *string) {
	s.internal.Name = name
}

func (s *Spec) Location(location *string) {
	s.internal.Location = location
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.VirtualNetwork) bool {
	// return any([]func() bool{
	// 	// Both immutable?
	// 	// func() bool { return Name(s) == nil || local.Spec.Name != *Name(s) },
	// 	// func() bool { return Location(s) == nil || local.Spec.Location != *Location(s) },
	// })
	return false
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

func PrincipalID(s *Spec) *string {
	if s == nil || s.internal == nil || s.internal.IdentityProperties == nil || s.internal.PrincipalID == nil {
		return nil
	}
	return to.StringPtr(s.internal.PrincipalID.String())
}

// func TenantID(s *Spec) *uuid.UUID {
// 	if s == nil || s.identity == nil || s.identity.IdentityProperties == nil || s.identity.TenantID == nil {
// 		return nil
// 	}
// 	return s.identity.TenantID
// }
