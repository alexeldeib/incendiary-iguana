/*
Copyright 2019 Alexander Eldeib.
*/

package keyvaults

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-logr/logr"
	uuid "github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal keyvault.VaultsClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) keyvault.VaultsClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, keyvault.NewVaultsClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
func NewWithFactory(configuration *config.Config, factory factoryFunc) *Client {
	return &Client{
		config:  configuration,
		factory: factory,
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	c.internal = c.factory(local.Spec.SubscriptionID)
	return c.config.AuthorizeClientFromArgs(&c.internal.Client)
}

// Ensure creates or updates a keyvault in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) error {
	vault, err := c.convert(obj)
	if err != nil {
		return err
	}
	// TODO(ace): handle location/name changes? via status somehow
	tenantId, err := uuid.FromString(vault.Spec.TenantID)
	if err != nil {
		return err
	}

	// TODO(ace): use spec here
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

	if _, err := c.internal.CreateOrUpdate(ctx, vault.Spec.ResourceGroup, vault.Spec.Name, opts); err != nil {
		return err
	}

	if err := c.SetStatus(ctx, vault); err != nil {
		return err
	}

	return nil
}

// Get returns a keyvault.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (keyvault.Vault, error) {
	local, err := c.convert(obj)
	if err != nil {
		return keyvault.Vault{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// Delete handles deletion of a keyvault and returns its provisioning state.
func (c *Client) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	response, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil && !response.IsHTTPStatus(http.StatusNotFound) {
		return err
	}
	return nil
}

func (c *Client) SetStatus(ctx context.Context, local *azurev1alpha1.Keyvault) error {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	local.Status.ID = remote.ID
	return err
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.Keyvault, error) {
	local, ok := obj.(*azurev1alpha1.Keyvault)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
