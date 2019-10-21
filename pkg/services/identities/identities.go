/*
Copyright 2019 Alexander Eldeib.
*/

package identities

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal msi.UserAssignedIdentitiesClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) msi.UserAssignedIdentitiesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, msi.NewUserAssignedIdentitiesClient)
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
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	c.internal = c.factory(local.Spec.SubscriptionID)
	return c.config.AuthorizeClientFromArgs(&c.internal.Client)
}

// Ensure creates or updates a managed identity in an idempotent manner.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return err
	}

	if found {
		return nil
	}

	spec := NewSpec()
	spec.Set(
		Location(&local.Spec.Location),
	)

	if _, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec.Build()); err != nil {
		return err
	}

	return nil
}

// Get returns a managed identity.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (msi.Identity, error) {
	local, err := c.convert(obj)
	if err != nil {
		return msi.Identity{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// Delete handles deletion of a managed identity.
func (c *Client) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	response, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil && !response.IsHTTPStatus(http.StatusNotFound) {
		return err
	}
	// remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	// found := !remote.IsHTTPStatus(http.StatusNotFound)
	// c.SetStatus(local, remote)
	// if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
	// 	return false, nil
	// }
	// return found, err
	return nil
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.Identity, remote msi.Identity) {
	local.Status.ID = remote.ID
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.Identity, error) {
	local, ok := obj.(*azurev1alpha1.Identity)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
