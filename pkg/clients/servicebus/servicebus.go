/*
Copyright 2019 Alexander Eldeib.
*/

package servicebus

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/servicebus/mgmt/2017-04-01/servicebus"
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
	internal   servicebus.NamespacesClient
	config     *config.Config
	kubeclient *ctrl.Client
	scheme     *runtime.Scheme
}

type factoryFunc func(subscriptionID string) servicebus.NamespacesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config, kubeclient *ctrl.Client, scheme *runtime.Scheme) *Client {
	return NewWithFactory(configuration, kubeclient, servicebus.NewNamespacesClient, scheme)
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

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
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

// Get returns a service bus.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (servicebus.SBNamespace, error) {
	local, err := c.convert(obj)
	if err != nil {
		return servicebus.SBNamespace{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// ListKeys returns an array of keys for a storage account.
func (c *Client) ListKeys(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) (map[string][]byte, error) {
	keys, err := c.internal.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name, "RootManageSharedAccessKey")
	if err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	if keys.PrimaryKey != nil {
		result[*local.Spec.PrimaryKey] = []byte(*keys.PrimaryKey)
	}
	if keys.SecondaryKey != nil {
		result[*local.Spec.SecondaryKey] = []byte(*keys.SecondaryKey)
	}

	if keys.PrimaryConnectionString != nil {
		result[*local.Spec.PrimaryConnectionString] = []byte(*keys.PrimaryConnectionString)
	}

	if keys.SecondaryConnectionString != nil {
		result[*local.Spec.SecondaryConnectionString] = []byte(*keys.SecondaryConnectionString)
	}
	return result, nil
}

func (c *Client) SyncSecrets(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) error {
	spew.Dump("listing keys")
	if local.Spec.TargetSecret == nil {
		return nil
	}
	keys, err := c.internal.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name, "RootManageSharedAccessKey")
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

		if local.Spec.PrimaryConnectionString != nil {
			if keys.PrimaryConnectionString != nil {
				targetSecret.Data[*local.Spec.PrimaryConnectionString] = []byte(*keys.PrimaryConnectionString)
			} else {
				final = multierror.Append(final, errors.New("expected primary key but found nil"))
			}

		}

		if local.Spec.SecondaryConnectionString != nil {
			if keys.SecondaryConnectionString != nil {
				targetSecret.Data[*local.Spec.SecondaryConnectionString] = []byte(*keys.SecondaryConnectionString)
			} else {
				final = multierror.Append(final, errors.New("expected primary key but found nil"))
			}

		}

		return final.ErrorOrNil()
	})
	spew.Dump("returning listkey")

	return err
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
	// Care about 400 and 5xx, not 404.
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.SBNamespaceProperties != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) bool {
	return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
}

func (c *Client) NeedsUpdate(local *azurev1alpha1.ServiceBusNamespace, remote servicebus.SBNamespace) bool {
	if remote.Sku != nil {
		if !strings.EqualFold(string(local.Spec.SKU.Name), string(remote.Sku.Name)) {
			spew.Dump("changed sku name")
			return true
		}
		if !strings.EqualFold(string(local.Spec.SKU.Tier), string(remote.Sku.Tier)) {
			spew.Dump("changed sku tier")
			return true
		}
		if remote.Sku.Capacity != nil && local.Spec.SKU.Capacity != *remote.Sku.Capacity {
			spew.Dump("changed capacity")
			return true
		}
	}
	if remote.Location != nil && strings.EqualFold(*remote.Location, local.Spec.Location) {
		spew.Dump("changed Location")
		return true
	}
	return false
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.ServiceBusNamespace, error) {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
