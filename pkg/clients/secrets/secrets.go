/*
Copyright 2019 Alexander Eldeib.
*/

package secrets

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

var _ Client = &client{}

// Client is the interface for Keyvault secrets. Defined for test mocks.
type Client interface {
	Get(context.Context, *azurev1alpha1.Secret) (keyvault.SecretBundle, error)
	// Set(context.Context, *azurev1alpha1.Secret) (keyvault.SecretBundle, error)
	Delete(context.Context, *azurev1alpha1.Secret) error
}

type client struct {
	internal keyvault.BaseClient
}

func New(configuration config.Config) (Client, error) {
	kvclient := keyvault.New()
	authorizer, err := configuration.GetKeyvaultAuthorizer()
	if err != nil {
		return nil, nil
	}
	kvclient.Authorizer = authorizer
	return &client{internal: kvclient}, nil
}

// Get gets a secret from Keyvault.
func (c *client) Get(ctx context.Context, secret *azurev1alpha1.Secret) (keyvault.SecretBundle, error) {
	vault := fmt.Sprintf("https://%s.%s", secret.Spec.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
	return c.internal.GetSecret(ctx, vault, secret.Spec.Name, "")
}

// // Set sets a secret in Keyvault. If a desired secret must be generated, this function will be called to
// // handle upload to Keyvault.
// func (c *client) Set(ctx context.Context, secret *azurev1alpha1.Secret) (keyvault.SecretBundle, error) {

// }

// Delete deletes a secret from Keyvault.
func (c *client) Delete(ctx context.Context, secret *azurev1alpha1.Secret) error {
	_, err := c.internal.DeleteSecret(ctx, secret.Spec.Vault, secret.Spec.Name)
	return err
}
