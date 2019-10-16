/*
Copyright 2019 Alexander Eldeib.
*/

package sqlfirewallrules

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2015-05-01-preview/sql"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal sql.FirewallRulesClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) sql.FirewallRulesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, sql.NewFirewallRulesClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
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

// Ensure creates or updates a SQL server in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}

	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Server, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return err
	}

	// Wrap, check status, and exit early if appropriate
	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		// TODO(ace): this is not checking whether the secret needs to be updated
		// TODO(ace): this should be an extension point to gracefully handle immutable updates
		// if !spec.NeedsUpdate(local) {
		// 	return nil
		// }
	} else {
		spec = NewSpec()
	}

	// Overlay new properties over old/default spec
	spec.Set(
		Start(&local.Spec.Start),
		End(&local.Spec.End),
	)

	_, err = c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Server, local.Spec.Name, spec.Build())
	return err
}

// Get returns a SQL server.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (sql.FirewallRule, error) {
	local, err := c.convert(obj)
	if err != nil {
		return sql.FirewallRule{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Server, local.Spec.Name)
}

// Delete handles deletion of a SQL server.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	_, err = c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Server, local.Spec.Name)
	return nil
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.SQLFirewallRule, remote sql.FirewallRule) {
	local.Status.ID = remote.ID
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.SQLFirewallRule, error) {
	local, ok := obj.(*azurev1alpha1.SQLFirewallRule)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
