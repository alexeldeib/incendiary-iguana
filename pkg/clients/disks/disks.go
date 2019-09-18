/*
Copyright 2019 Alexander Eldeib.
*/

package disks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal compute.DisksClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) compute.DisksClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, compute.NewDisksClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
func NewWithFactory(configuration *config.Config, factory factoryFunc) *Client {
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

// Delete handles deletion of a disk.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.VM) (bool, error) {
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, fmt.Sprintf("%s_%s_%s_osdisk", local.Spec.SubscriptionID, local.Spec.ResourceGroup, local.Spec.Name))
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && !found {
		return false, nil
	}
	return found, err
}
