/*
Copyright 2019 Alexander Eldeib.
*/

package servicebus

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/servicebus/mgmt/2017-04-01/servicebus"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/davecgh/go-spew/spew"
)

type Client struct {
	factory  factoryFunc
	internal servicebus.NamespacesClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) servicebus.NamespacesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, servicebus.NewNamespacesClient)
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
func (c *Client) ForSubscription(subID string) error {
	c.internal = c.factory(subID)
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	if found {
		if c.Done(ctx, local) {
			if !c.NeedsUpdate(local, remote) {
				return true, nil
			}
		} else {
			spew.Dump("not done")
			return false, nil
		}
	}

	spec := servicebus.SBNamespace{
		Location: &local.Spec.Location,
		Sku: &servicebus.SBSku{
			Name:     servicebus.SkuName(local.Spec.SKU.Name),
			Tier:     servicebus.SkuTier(local.Spec.SKU.Tier),
			Capacity: &local.Spec.SKU.Capacity,
		},
	}

	if _, err = c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
		return false, err
	}
	return false, nil
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) (servicebus.SBNamespace, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) (bool, error) {
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
		return false, nil
	}
	return found, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.ServiceBusNamespace, remote servicebus.SBNamespace) {
	spew.Dump(remote.SBNamespaceProperties)
	spew.Dump(remote.Location)
	// Care about 400 and 5xx, not 404.
	local.Status.ID = remote.ID
	if remote.SBNamespaceProperties != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) bool {
	return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
}

// InProgress
func (c *Client) InProgress(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) bool {
	return local.Status.ProvisioningState != nil
}

func (c *Client) NeedsUpdate(local *azurev1alpha1.ServiceBusNamespace, remote servicebus.SBNamespace) bool {
	if remote.Sku != nil {

	}
	if servicebus.SkuName(local.Spec.SKU.Name) != remote.Sku.Name {
		spew.Dump("changed SkuName")
		return true
	}
	if servicebus.SkuTier(local.Spec.SKU.Tier) != remote.Sku.Tier {
		spew.Dump("changed SkuTier")
		return true
	}
	if remote.Sku.Capacity == nil && local.Spec.SKU.Capacity != *remote.Sku.Capacity {
		spew.Dump("changed Capacity")
		return true
	}
	if remote.Location != nil {
		cleaned := strings.ToLower(strings.Replace(*remote.Location, " ", "", -1))
		if cleaned != local.Spec.Location {
			spew.Dump("changed Location")
			return true
		}
	}
	return false
}

func (c *Client) TryAuthorize(ctx context.Context, obj runtime.Object) error {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.ForSubscription(local.Spec.SubscriptionID)
}

func (c *Client) TryEnsure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Ensure(ctx, local)
}

func (c *Client) TryDelete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Delete(ctx, local)
}
