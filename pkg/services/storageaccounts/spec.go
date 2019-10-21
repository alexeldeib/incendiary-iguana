/*
Copyright 2019 Alexander Eldeib.
*/

package storageaccounts

/*
Copyright 2019 Alexander Eldeib.
*/

import (
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/google/go-cmp/cmp"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/clientutil"
)

type Spec struct {
	internal *storage.Account
}

func NewSpec() *Spec {
	return &Spec{
		internal: &storage.Account{},
	}
}

func NewSpecWithRemote(remote *storage.Account) *Spec {
	return &Spec{
		internal: remote,
	}
}

func (s *Spec) Set(opts ...func(*Spec)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Spec) ForCreate() storage.AccountCreateParameters {
	result := storage.AccountCreateParameters{
		Sku: &storage.Sku{
			Kind: storage.Storage,
		},
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
	}
	if s.internal.Sku != nil {
		result.Sku = s.internal.Sku
	}
	result.Location = s.Location()
	return result
}

func (s *Spec) ForUpdate() storage.AccountUpdateParameters {
	result := storage.AccountUpdateParameters{
		Sku: &storage.Sku{
			Kind: storage.Storage,
		},
		AccountPropertiesUpdateParameters: &storage.AccountPropertiesUpdateParameters{},
	}
	if s.internal.Sku != nil {
		result.Sku = s.internal.Sku
	}
	return result
}

func Location(location *string) func(*Spec) {
	return func(s *Spec) {
		s.internal.Location = location
	}
}

func (s *Spec) NeedsUpdate(local *azurev1alpha1.PublicIP) bool {
	// lol
	return clientutil.Any([]func() bool{
		func() bool { return !cmp.Equal(s.Location(), &local.Spec.Location) },
	})
}

func (s *Spec) Location() *string {
	return s.internal.Location
}

func (s *Spec) ID() *string {
	return s.internal.ID
}

func (s *Spec) State() *string {
	if s.internal.AccountProperties == nil {
		return nil
	}
	return to.StringPtr(string(s.internal.AccountProperties.ProvisioningState))
}
