/*
Copyright 2019 Alexander Eldeib.
*/

package resourcegroups

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal resources.GroupsClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) resources.GroupsClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, resources.NewGroupsClient)
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

// Ensure creates or updates a resource group in an idempotent manner.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	remote, err := c.internal.Get(ctx, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		if c.Done(ctx, local) {
			if !spec.NeedsUpdate(local) {
				return true, nil
			}
		} else {
			return false, nil
		}
	} else {
		spec = NewSpec()
	}

	spec.Set(
		Name(local.Spec.Name),
		Location(local.Spec.Location),
	)

	_, err = c.internal.CreateOrUpdate(ctx, local.Spec.Name, spec.Build())
	return false, err
}

// Delete handles deletion of a resource groups.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	future, err := c.internal.Delete(ctx, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		resp := future.Response()
		if resp != nil && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusConflict {
			return false, err
		}
	}
	remote, err := c.internal.Get(ctx, local.Spec.Name)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
		return false, nil
	}
	return found, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.ResourceGroup, remote resources.Group) {
	local.Status.ID = remote.ID
	if remote.Properties != nil {
		local.Status.ProvisioningState = remote.Properties.ProvisioningState
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.ResourceGroup) bool {
	return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
}

func (c *Client) NeedsUpdate(local *azurev1alpha1.ResourceGroup, remote resources.Group) bool {
	if remote.Name != nil && local.Spec.Name != *remote.Name {
		return true
	}
	if remote.Location != nil && local.Spec.Location != *remote.Location {
		return true
	}
	return false
}

func LogRequest() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				fmt.Println(err)
			}
			dump, _ := httputil.DumpRequestOut(r, true)
			fmt.Println(string(dump))
			return r, err
		})
	}
}

func LogResponse() autorest.RespondDecorator {
	return func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			err := p.Respond(r)
			if err != nil {
				fmt.Println(err)
			}
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Println(string(dump))
			return err
		})
	}
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.ResourceGroup, error) {
	local, ok := obj.(*azurev1alpha1.ResourceGroup)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
