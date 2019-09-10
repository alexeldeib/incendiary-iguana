/*
Copyright 2019 Alexander Eldeib.
*/

package redis

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"

	"github.com/davecgh/go-spew/spew"
	"github.com/sanity-io/litter"
)

type Client struct {
	factory  factoryFunc
	internal redis.Client
	config   *config.Config
}

type factoryFunc func(subscriptionID string) redis.Client

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, redis.NewClient)
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
	// c.internal.RequestInspector = LogRequest()
	// c.internal.ResponseInspector = LogResponse()
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a resource group in an idempotent manner.
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.Redis) (bool, error) {
	fmt.Printf("Inside ensure\n")
	found, err := c.SetStatus(ctx, local)
	if err != nil {
		return false, err
	}
	litter.Dump(local)

	fmt.Printf("comparing generation\n")
	if found && !c.Done(ctx, local) {
		spew.Dump("not done")
		return false, nil
	}

	spec := redis.CreateParameters{
		Location: &local.Spec.Location,
		CreateProperties: &redis.CreateProperties{
			EnableNonSslPort: &local.Spec.EnableNonSslPort,
			Sku: &redis.Sku{
				Name:     redis.SkuName(local.Spec.Sku.Name),
				Family:   redis.SkuFamily(local.Spec.Sku.Family),
				Capacity: &local.Spec.Sku.Capacity,
			},
		},
	}

	fmt.Printf("creating cache\n")
	if _, err := c.internal.Create(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
		fmt.Printf("create err\n")
		// if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusConflict {
		// fmt.Printf("create conflict err\n")
		return false, err
		// }
	}
	fmt.Printf("setting generation\n")
	local.Status.ObservedGeneration = local.ObjectMeta.GetGeneration()
	return false, nil
}

// Get returns a resource group.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.Redis) (redis.ResourceType, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// Delete handles deletion of a resource groups.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.Redis) (bool, error) {
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		resp := future.Response()
		if resp != nil && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusConflict {
			return false, err
		}
	}
	return c.SetStatus(ctx, local)
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(ctx context.Context, local *azurev1alpha1.Redis) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	spew.Config.MaxDepth = 3
	spew.Dump("inside remote")
	spew.Dump(remote.Properties)
	// Care about 400 and 5xx, not 404.
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && !remote.HasHTTPStatus(http.StatusNotFound, http.StatusConflict) {
		fmt.Printf("not found\n")
		return found, err
	}

	local.Status.ID = remote.ID
	local.Status.ProvisioningState = ""
	if remote.Properties != nil {
		local.Status.ProvisioningState = string(remote.Properties.ProvisioningState)
	}
	return found, nil
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.Redis) bool {
	return local.Status.ProvisioningState == "Succeeded"
}

// InProgress
func (c *Client) InProgress(ctx context.Context, local *azurev1alpha1.Redis) bool {
	return local.Status.ProvisioningState != ""
}

// TODO(ace): improve naming and the structure of this pattern across all gvks
func (c *Client) TryAuthorize(ctx context.Context, obj runtime.Object) error {
	local, ok := obj.(*azurev1alpha1.Redis)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.ForSubscription(local.Spec.SubscriptionID)
}

func (c *Client) TryEnsure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.Redis)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Ensure(ctx, local)
}

func (c *Client) TryDelete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.Redis)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Delete(ctx, local)
}
