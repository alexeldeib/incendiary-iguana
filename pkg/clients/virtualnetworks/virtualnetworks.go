/*
Copyright 2019 Alexander Eldeib.
*/

package virtualnetworks

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const expand string = ""

// Type assertion for interface/implementation
var _ Client = &client{}

// Client is the interface for Azure resource groups. Defined for test mocks.
type Client interface {
	ForSubscription(string) error
	Ensure(context.Context, *azurev1alpha1.VirtualNetwork, network.VirtualNetwork) error
	Get(context.Context, *azurev1alpha1.VirtualNetwork) (network.VirtualNetwork, error)
	Delete(context.Context, *azurev1alpha1.VirtualNetwork) error
}

type client struct {
	factory  factoryFunc
	internal network.VirtualNetworksClient
	config   config.Config
}

type factoryFunc func(subscriptionID string) network.VirtualNetworksClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration config.Config) Client {
	return NewWithFactory(configuration, network.NewVirtualNetworksClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
func NewWithFactory(configuration config.Config, factory factoryFunc) Client {
	return &client{
		config:  configuration,
		factory: factory,
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *client) ForSubscription(subID string) error {
	c.internal = c.factory(subID)
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *client) Ensure(ctx context.Context, local *azurev1alpha1.VirtualNetwork, remote network.VirtualNetwork) error {
	spec := remote
	spec.Location = &local.Spec.Location
	spec.VirtualNetworkPropertiesFormat = &network.VirtualNetworkPropertiesFormat{
		AddressSpace: &network.AddressSpace{
			AddressPrefixes: &local.Spec.Addresses,
		},
	}
	_, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec)
	return err
}

// Get returns a virtual network.
func (c *client) Get(ctx context.Context, local *azurev1alpha1.VirtualNetwork) (network.VirtualNetwork, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
}

// Delete handles deletion of a virtual network.
func (c *client) Delete(ctx context.Context, local *azurev1alpha1.VirtualNetwork) error {
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return err
		}
	}
	return nil
}