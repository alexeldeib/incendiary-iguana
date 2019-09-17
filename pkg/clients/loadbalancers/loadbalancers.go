/*
Copyright 2019 Alexander Eldeib.
*/

package loadbalancers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/davecgh/go-spew/spew"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const expand string = ""

type Client struct {
	factory  factoryFunc
	internal network.LoadBalancersClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) network.LoadBalancersClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, network.NewLoadBalancersClient)
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
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.LoadBalancer) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
	found := !remote.HasHTTPStatus(http.StatusNotFound, http.StatusConflict)
	c.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		if c.Done(ctx, local) {
			return true, nil
		}
		spew.Dump("not done")
		return false, nil
	} else {
		spec = NewSpec()
	}

	spec.Set(
		Name(local.Spec.Name),
		Location(local.Spec.Location),
		Frontends(local.Spec.Frontends),
		Backends(local.Spec.BackendPools),
	)

	_, err = c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec.Build())
	return false, err
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.LoadBalancer) (network.LoadBalancer, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.LoadBalancer) (bool, error) {
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
		return false, nil
	}
	return found, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.LoadBalancer, remote network.LoadBalancer) {
	local.Status.ID = remote.ID
	if remote.LoadBalancerPropertiesFormat != nil {
		local.Status.ProvisioningState = remote.LoadBalancerPropertiesFormat.ProvisioningState
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.LoadBalancer) bool {
	if local.Status.ProvisioningState == nil || *local.Status.ProvisioningState != "Succeeded" {
		return false
	}
	return true
}

func (c *Client) TryAuthorize(ctx context.Context, obj runtime.Object) error {
	local, ok := obj.(*azurev1alpha1.LoadBalancer)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.ForSubscription(local.Spec.SubscriptionID)
}

func (c *Client) TryEnsure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.LoadBalancer)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Ensure(ctx, local)
}

func (c *Client) TryDelete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.LoadBalancer)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Delete(ctx, local)
}
