/*
Copyright 2019 Alexander Eldeib.
*/

package secrets

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
)

type SecretService struct {
	DNSSuffix string
	Client    keyvault.BaseClient
}

func NewSecretService(authorizer autorest.Authorizer, opts ...func(*SecretService)) *SecretService {
	return &SecretService{
		DNSSuffix: azure.PublicCloud.KeyVaultDNSSuffix,
		Client:    clients.NewSecretsClient(authorizer),
	}
}

func DNSSuffix(suffix string) func(*SecretService) {
	return func(s *SecretService) {
		s.DNSSuffix = suffix
	}
}

func (s *SecretService) Get(ctx context.Context, vault, name string) (result keyvault.SecretBundle, err error) {
	vaultHost := fmt.Sprintf("https://%s.%s", vault, s.DNSSuffix)
	return s.Client.GetSecret(ctx, vaultHost, name, "")
}

func (s *SecretService) GetCertificate(ctx context.Context, vault, name string) (result keyvault.CertificateBundle, err error) {
	vaultHost := fmt.Sprintf("https://%s.%s", vault, s.DNSSuffix)
	return s.Client.GetCertificate(ctx, vaultHost, name, "")
}
