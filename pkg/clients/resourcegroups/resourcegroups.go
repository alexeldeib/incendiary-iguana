/*
Copyright 2019 Alexander Eldeib.
*/

package resourcegroups

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"

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
func (c *Client) ForSubscription(subID string) error {
	c.internal = c.factory(subID)
	c.internal.RequestInspector = LogRequest()
	c.internal.ResponseInspector = LogResponse()
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a resource group in an idempotent manner.
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.ResourceGroup) error {
	spec := resources.Group{
		Location: &local.Spec.Location,
	}

	if _, err := c.internal.CreateOrUpdate(ctx, local.Spec.Name, spec); err != nil {
		return err
	}

	if _, err := c.SetStatus(ctx, local); err != nil {
		return err
	}

	if !c.Done(ctx, local) {
		return errors.New("not finished reconciling, requeueing")
	}

	return nil
}

// Get returns a resource group.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.ResourceGroup) (resources.Group, error) {
	return c.internal.Get(ctx, local.Spec.Name)
}

// Delete handles deletion of a resource groups.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.ResourceGroup) (bool, error) {
	future, err := c.internal.Delete(ctx, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	return c.SetStatus(ctx, local)
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(ctx context.Context, local *azurev1alpha1.ResourceGroup) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.Name)
	// Care about 400 and 5xx, not 404.
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && found {
		return found, err
	}

	local.Status.ID = remote.ID
	if local.Status.ProvisioningState == nil {
		local.Status.ProvisioningState = new(string)
	}
	if remote.Properties != nil && remote.Properties.ProvisioningState != nil {
		*local.Status.ProvisioningState = *remote.Properties.ProvisioningState
	}
	return found, nil
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.ResourceGroup) bool {
	if local.Status.ProvisioningState == nil || *local.Status.ProvisioningState != "Succeeded" {
		return false
	}
	return true
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
