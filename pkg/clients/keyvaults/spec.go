/*
Copyright 2019 Alexander Eldeib.
*/

package keyvaults

// import (
// 	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
// 	"github.com/google/go-cmp/cmp"

// 	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
// 	"github.com/alexeldeib/incendiary-iguana/pkg/services/clientutil"
// )

// type Spec struct {
// 	internal *keyvault.Vault
// }

// func NewSpec() *Spec {
// 	return &Spec{
// 		internal: &keyvault.Vault{},
// 	}
// }

// func NewSpecWithRemote(remote *keyvault.Vault) *Spec {
// 	return &Spec{
// 		internal: remote,
// 	}
// }

// func (s *Spec) Set(opts ...func(*Spec)) {
// 	for _, opt := range opts {
// 		opt(s)
// 	}
// }

// func (s *Spec) Build() keyvault.Vault {
// 	return *s.internal
// }

// func Name(name string) func(*Spec) {
// 	return func(s *Spec) {
// 		s.internal.Name = &name
// 	}
// }

// func Location(location string) func(*Spec) {
// 	return func(s *Spec) {
// 		s.internal.Location = &location
// 	}
// }

// func (s *Spec) NeedsUpdate(local *azurev1alpha1.ResourceGroup) bool {
// 	return clientutil.Any([]func() bool{
// 		func() bool { return !cmp.Equal(s.Name(), &local.Spec.Name) },
// 		func() bool { return !cmp.Equal(s.Location(), &local.Spec.Location) },
// 	})
// }

// func (s *Spec) Name() *string {
// 	return s.internal.Name
// }

// func (s *Spec) Location() *string {
// 	return s.internal.Location
// }

// func (s *Spec) ID() *string {
// 	return s.internal.ID
// }
