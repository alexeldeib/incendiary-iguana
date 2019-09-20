/*
Copyright 2019 Alexander Eldeib.
*/

package publicips

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const expand string = ""

type Client struct {
	factory  factoryFunc
	internal network.PublicIPAddressesClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) network.PublicIPAddressesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, network.NewPublicIPAddressesClient)
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
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	// TODO(ace): use spec.Set() pattern from other packages
	spec := network.PublicIPAddress{
		Location: &local.Spec.Location,
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuNameStandard,
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Static,
		},
	}
	if local.Spec.SKU != nil {
		spec.Sku.Name = network.PublicIPAddressSkuName(*local.Spec.SKU)
	}

	if future, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusConflict {
			return false, err
		}
		return false, nil
	}

	if _, err := c.SetStatus(ctx, local); err != nil {
		return false, err
	}

	return c.Done(ctx, local), nil
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (network.PublicIPAddress, error) {
	local, err := c.convert(obj)
	if err != nil {
		return network.PublicIPAddress{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	return c.SetStatus(ctx, local)
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(ctx context.Context, local *azurev1alpha1.PublicIP) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
	// Care about 400 and 5xx, not 404.
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && found {
		if remote.IsHTTPStatus(http.StatusConflict) {
			return found, nil
		}
		return found, err
	}

	local.Status.ID = remote.ID
	if remote.PublicIPAddressPropertiesFormat != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
	return found, nil
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.PublicIP) bool {
	if local.Status.ProvisioningState == nil || *local.Status.ProvisioningState != "Succeeded" {
		return false
	}
	return true
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.PublicIP, error) {
	local, ok := obj.(*azurev1alpha1.PublicIP)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
