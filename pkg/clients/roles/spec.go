/*
Copyright 2019 Alexander Eldeib.
*/

package roles

import (
	"github.com/Azure/azure-sdk-for-go/profiles/latest/msi/mgmt/msi"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type Spec struct {
	internal *msi.Identity
}

func NewSpec() *Spec {
	return &Spec{
		internal: &msi.Identity{},
	}
}

func NewSpecWithRemote(remote *msi.Identity) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Build() msi.Identity {
	return *s.internal
}

// func (s *Spec) Scope(scope *string) {
// 	s.initialize(
// 		[]func() bool{
// 			func() bool { return s.internal.Properties == nil },
// 		},
// 		[]func(){
// 			func() { s.internal.Properties = &authorization.RoleAssignmentPropertiesWithScope{} },
// 		},
// 	)
// 	s.internal.Properties.Scope = scope
// }

// func (s *Spec) PrincipalID(principalID *string) {
// 	s.initialize(
// 		[]func() bool{
// 			func() bool { return s.internal.Properties == nil },
// 		},
// 		[]func(){
// 			func() { s.internal.Properties = &authorization.RoleAssignmentPropertiesWithScope{} },
// 		},
// 	)
// 	s.internal.Properties.PrincipalID = principalID
// }

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

func (s *Spec) initialize(detectors []func() bool, remediators []func()) {
	for idx, f := range detectors {
		if f() {
			remediators[idx]()
		}
	}
}
