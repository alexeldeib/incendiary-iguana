/*
Copyright 2019 Alexander Eldeib.
*/

package subnets

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const expand string = ""

type Client struct {
	factory  factoryFunc
	internal network.SubnetsClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) network.SubnetsClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, network.NewSubnetsClient)
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

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name, expand)
	c.SetStatus(local, remote)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && found {
		return found, err
	}

	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		if c.Done(ctx, local) {
			if !spec.NeedsUpdate(local) {
				return true, nil
			}
		} else {
			spew.Dump("not done")
			return false, nil
		}
	} else {
		spec = NewSpec()
	}

	spec.Name(local.Spec.Name)
	spec.Address(local.Spec.Subnet)

	if _, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name, spec.Build()); err != nil {
		return false, err
	}

	return false, nil
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (network.Subnet, error) {
	local, err := c.convert(obj)
	if err != nil {
		return network.Subnet{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name, expand)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name, expand)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
		return false, nil
	}
	return found, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.Subnet, remote network.Subnet) {
	local.Status.ID = remote.ID
	if remote.SubnetPropertiesFormat != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.Subnet) bool {
	if local.Status.ProvisioningState == nil || *local.Status.ProvisioningState != "Succeeded" {
		return false
	}
	return true
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.Subnet, error) {
	local, ok := obj.(*azurev1alpha1.Subnet)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
