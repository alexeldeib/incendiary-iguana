/*
Copyright 2019 Alexander Eldeib.
*/

package keyvaults

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	uuid "github.com/satori/go.uuid"
)

type Client struct {
	factory  factoryFunc
	internal keyvault.VaultsClient
	config   config.Config
}

type factoryFunc func(subscriptionID string) keyvault.VaultsClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration config.Config) *Client {
	return NewWithFactory(configuration, keyvault.NewVaultsClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
func NewWithFactory(configuration config.Config, factory factoryFunc) *Client {
	return &Client{
		config:  configuration,
		factory: factory,
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(subID string) error {
	c.internal = c.factory(subID)
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a keyvault in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, vault *azurev1alpha1.Keyvault) error {
	// TODO(ace): handle location/name changes? via status somehow
	tenantId, err := uuid.FromString(vault.Spec.TenantID)
	if err != nil {
		return err
	}
	opts := keyvault.VaultCreateOrUpdateParameters{
		Properties: &keyvault.VaultProperties{
			TenantID:       &tenantId,
			AccessPolicies: &[]keyvault.AccessPolicyEntry{},
			Sku: &keyvault.Sku{
				Family: to.StringPtr("A"),
				Name:   keyvault.Standard,
			},
		},
		Location: &vault.Spec.Location,
	}
	_, err = c.internal.CreateOrUpdate(ctx, vault.Spec.ResourceGroup, vault.Spec.Name, opts)
	return err
}

// Get returns a keyvault.
func (c *Client) Get(ctx context.Context, vault *azurev1alpha1.Keyvault) (keyvault.Vault, error) {
	return c.internal.Get(ctx, vault.Spec.ResourceGroup, vault.Spec.Name)
}

// Delete handles deletion of a keyvault and returns its provisioning state.
func (c *Client) Delete(ctx context.Context, vault *azurev1alpha1.Keyvault) error {
	response, err := c.internal.Delete(ctx, vault.Spec.ResourceGroup, vault.Spec.Name)
	if err != nil && !response.IsHTTPStatus(http.StatusNotFound) {
		return err
	}
	return nil
}
