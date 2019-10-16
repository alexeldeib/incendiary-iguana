/*
Copyright 2019 Alexander Eldeib.
*/

package disks

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/controllers"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

var _ controllers.AsyncClient = &Client{}

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
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	c.internal = c.factory(local.Spec.SubscriptionID)
	return c.config.AuthorizeClientFromArgs(&c.internal.Client)
}

// Ensure handles reconciliation of a disk.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	_, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	return false, errors.New("not implemented")
}

// Delete handles deletion of a disk.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}

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

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.VM, error) {
	local, ok := obj.(*azurev1alpha1.VM)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
