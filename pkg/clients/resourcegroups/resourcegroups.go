/*
Copyright 2019 Alexander Eldeib.
*/

package resourcegroups

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal resources.GroupsClient
	config   config.Config
}

type factoryFunc func(subscriptionID string) resources.GroupsClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration config.Config) *Client {
	return NewWithFactory(configuration, resources.NewGroupsClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
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

// Ensure creates or updates a resource group in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, resourceGroup *azurev1alpha1.ResourceGroup) (resources.Group, error) {
	// Check for existence of Resource group. We only care about location and name.
	// TODO(ace): handle location/name changes? via status somehow
	group, err := c.internal.Get(ctx, resourceGroup.Spec.Name)
	if err != nil && !group.IsHTTPStatus(http.StatusNotFound) {
		return resources.Group{}, err
	}
	spec := resources.Group{
		Location: &resourceGroup.Spec.Location,
	}
	group, err = c.internal.CreateOrUpdate(ctx, resourceGroup.Spec.Name, spec)
	return group, err
}

// Get returns a resource group and sets its provisioning state.
func (c *Client) Get(ctx context.Context, resourceGroup *azurev1alpha1.ResourceGroup) (resources.Group, error) {
	return c.internal.Get(ctx, resourceGroup.Spec.Name)
}

// Delete handles deletion of a resource groups and sets its provisioning state.
func (c *Client) Delete(ctx context.Context, resourceGroup *azurev1alpha1.ResourceGroup) (string, error) {
	future, err := c.internal.Delete(ctx, resourceGroup.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return "", err
		}
	}
	return future.Status(), nil
}
