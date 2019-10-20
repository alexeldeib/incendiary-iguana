/*
Copyright 2019 Alexander Eldeib.
*/

package redis

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"

	"github.com/davecgh/go-spew/spew"
)

type Client struct {
	factory    factoryFunc
	internal   redis.Client
	config     *config.Config
	kubeclient *ctrl.Client
	scheme     *runtime.Scheme
}

type factoryFunc func(subscriptionID string) redis.Client

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config, kubeclient *ctrl.Client, scheme *runtime.Scheme) *Client {
	return NewWithFactory(configuration, kubeclient, redis.NewClient, scheme)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
func NewWithFactory(configuration *config.Config, kubeclient *ctrl.Client, factory factoryFunc, scheme *runtime.Scheme) *Client {
	return &Client{
		config:     configuration,
		factory:    factory,
		kubeclient: kubeclient,
		scheme:     scheme,
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

// Ensure creates or updates a redis cache in an idempotent manner.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	if found {
		if c.Done(ctx, local) {
			if c.kubeclient != nil {
				if err := c.SyncSecrets(ctx, local); err != nil {
					return false, err
				}
			}
			if !c.NeedsUpdate(local, remote) {
				return true, nil
			}
		} else {
			spew.Dump("not done")
			return false, nil
		}
	}

	// TODO(ace): spec.Set()
	spec := redis.CreateParameters{
		Location: &local.Spec.Location,
		CreateProperties: &redis.CreateProperties{
			EnableNonSslPort: &local.Spec.EnableNonSslPort,
			Sku: &redis.Sku{
				Name:     redis.SkuName(local.Spec.SKU.Name),
				Family:   redis.SkuFamily(local.Spec.SKU.Family),
				Capacity: &local.Spec.SKU.Capacity,
			},
		},
	}

	if _, err = c.internal.Create(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
		return false, err
	}

	return false, nil
}

func (c *Client) SyncSecrets(ctx context.Context, local *azurev1alpha1.Redis) error {
	if local.Spec.PrimaryKey == nil && local.Spec.SecondaryKey == nil {
		return nil
	}
	keys, err := c.internal.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		spew.Dump("err listing keys")
		return err
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *local.Spec.TargetSecret,
			Namespace: local.ObjectMeta.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, *c.kubeclient, targetSecret, func() error {
		spew.Dump("set primary")
		var final *multierror.Error

		if targetSecret.Data == nil {
			targetSecret.Data = map[string][]byte{}
		}

		// if local.ObjectMeta.UID != "" {
		// 	final = multierror.Append(final, controllerutil.SetControllerReference(local, targetSecret, c.scheme))
		// }

		if local.Spec.PrimaryKey != nil {
			if keys.PrimaryKey != nil {
				targetSecret.Data[*local.Spec.PrimaryKey] = []byte(*keys.PrimaryKey)
			} else {
				final = multierror.Append(final, errors.New("expected primary key but found nil"))
			}
		}

		if local.Spec.SecondaryKey != nil {
			if keys.SecondaryKey != nil {
				targetSecret.Data[*local.Spec.SecondaryKey] = []byte(*keys.SecondaryKey)
			} else {
				final = multierror.Append(final, errors.New("expected primary key but found nil"))
			}

		}

		return final.ErrorOrNil()
	})
	spew.Dump("returning listkey")
	return err
}

// Delete handles deletion of a resource groups.
func (c *Client) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		resp := future.Response()
		if resp != nil && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusConflict {
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
func (c *Client) SetStatus(local *azurev1alpha1.Redis, remote redis.ResourceType) {
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = ""
	if remote.Properties != nil {
		local.Status.ProvisioningState = string(remote.Properties.ProvisioningState)
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.Redis) bool {
	return local.Status.ProvisioningState == "Succeeded"
}

func (c *Client) NeedsUpdate(local *azurev1alpha1.Redis, remote redis.ResourceType) bool {
	if remote.Sku != nil {
		spew.Dump("local")
		spew.Dump(local.Spec.SKU)
		spew.Dump("remote")
		spew.Dump(remote.Sku)
		if !strings.EqualFold(string(local.Spec.SKU.Name), string(remote.Sku.Name)) {
			spew.Dump("changed SkuName")
			return true
		}
		if !strings.EqualFold(string(local.Spec.SKU.Family), string(remote.Sku.Family)) {
			spew.Dump("changed SkuFamily")
			return true
		}
		if remote.Sku.Capacity == nil && local.Spec.SKU.Capacity != *remote.Sku.Capacity {
			spew.Dump("changed Capacity")
			return true
		}
	}
	if remote.EnableNonSslPort != nil && *remote.EnableNonSslPort != local.Spec.EnableNonSslPort {
		spew.Dump("changed non ssl port")
		return true
	}
	if remote.Location != nil && strings.EqualFold(*remote.Location, local.Spec.Location) {
		spew.Dump("changed Location")
		return true
	}
	return false
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.Redis, error) {
	local, ok := obj.(*azurev1alpha1.Redis)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
