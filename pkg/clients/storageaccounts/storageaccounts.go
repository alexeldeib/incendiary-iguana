/*
Copyright 2019 Alexander Eldeib.
*/

package storageaccounts

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	config     *config.Config
	factory    factoryFunc
	internal   storage.AccountsClient
	kubeclient *ctrl.Client
	scheme     *runtime.Scheme
}

type factoryFunc func(subscriptionID string) storage.AccountsClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config, kubeclient *ctrl.Client, scheme *runtime.Scheme) *Client {
	return NewWithFactory(configuration, kubeclient, storage.NewAccountsClient, scheme)
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
func (c *Client) ForSubscription(obj runtime.Object) error {
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

	// Set status
	remote, err := c.internal.GetProperties(ctx, local.Spec.ResourceGroup, local.Spec.Name, "")
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return err
	}

	// Wrap, check status, and exit early if appropriate
	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		if err := c.SyncSecret(ctx, local); err != nil {
			return err
		}
		// TODO(ace): this should be an extension point to gracefully handle immutable updates
		// if c.Done(local)
		// if !spec.NeedsUpdate(local) {
		// 	return nil
		// }
	} else {
		spec = NewSpec()
	}

	// Overlay new properties over old/default spec
	spec.Set(
		Location(&local.Spec.Location),
	)

	if !found {
		if _, err = c.internal.Create(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec.ForCreate()); err != nil {
			return err
		}
		return nil
	}
	_, err = c.internal.Update(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec.ForUpdate())
	return err
}

// ListKeys returns a virtual network.
func (c *Client) ListKeys(ctx context.Context, local *azurev1alpha1.StorageAccount) (map[string][]byte, error) {
	keys, err := c.internal.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	// TODO(ace): think this might be safe? confirm.
	result[*local.Spec.PrimaryKey] = []byte(*(*keys.Keys)[0].Value)
	return result, nil
}

func (c *Client) SyncSecret(ctx context.Context, local *azurev1alpha1.StorageAccount) error {
	if local.Spec.TargetSecret == nil {
		return nil
	}

	keys, err := c.ListKeys(ctx, local)
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
		if targetSecret.Data == nil {
			targetSecret.Data = map[string][]byte{}
		}
		for key, val := range keys {
			targetSecret.Data[key] = []byte(val)
		}
		if local.ObjectMeta.UID != "" {
			if ownerErr := controllerutil.SetControllerReference(local, targetSecret, c.scheme); ownerErr != nil {
				return ownerErr
			}
		}
		return nil
	})

	return err
}

// Delete handles deletion of a SQL server.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.Name,
			Namespace: local.ObjectMeta.Namespace,
		},
	}

	// Get SQL Server secret
	if err = (*c.kubeclient).Delete(ctx, targetSecret); client.IgnoreNotFound(err) != nil {
		return err
	}

	_, err = c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	return err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.StorageAccount, remote storage.Account) {
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.AccountProperties != nil {
		local.Status.ProvisioningState = to.StringPtr(string(remote.AccountProperties.ProvisioningState))
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(local *azurev1alpha1.StorageAccount) bool {
	return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.StorageAccount, error) {
	local, ok := obj.(*azurev1alpha1.StorageAccount)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
