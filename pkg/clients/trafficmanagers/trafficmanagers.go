/*
Copyright 2019 Alexander Eldeib.
*/

package trafficmanagers

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/trafficmanager/mgmt/2018-04-01/trafficmanager"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

// Type assertion for interface/implementation
var _ Client = &client{}

// Client is the interface for Azure public ip addresses. Defined for test mocks.
type Client interface {
	ForSubscription(string) error
	Ensure(context.Context, *azurev1alpha1.TrafficManager, trafficmanager.Profile) error
	Get(context.Context, *azurev1alpha1.TrafficManager) (trafficmanager.Profile, error)
	Delete(context.Context, *azurev1alpha1.TrafficManager) error
}

type client struct {
	factory  factoryFunc
	internal trafficmanager.ProfilesClient
	config   config.Config
}

type factoryFunc func(subscriptionID string) trafficmanager.ProfilesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration config.Config) Client {
	return NewWithFactory(configuration, trafficmanager.NewProfilesClient)
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
func (c *client) Ensure(ctx context.Context, local *azurev1alpha1.TrafficManager, remote trafficmanager.Profile) error {
	spec := trafficmanager.Profile{}
	_, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec)
	return err
}

// Get returns a virtual network.
func (c *client) Get(ctx context.Context, local *azurev1alpha1.TrafficManager) (trafficmanager.Profile, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// Delete handles deletion of a virtual network.
func (c *client) Delete(ctx context.Context, local *azurev1alpha1.TrafficManager) error {
	_, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	return err
}
