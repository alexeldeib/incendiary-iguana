/*
Copyright 2019 Alexander Eldeib.
*/

package resources

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

// Type assertion for interface/implementation
var _ Client = &client{}

// Client is the interface for Azure resource groups. Defined for test mocks.
type Client interface {
	ForSubscription(string) error
	Exists(context.Context, string) (bool, error)
}

type client struct {
	factory  factoryFunc
	internal resources.Client
	config   config.Config
}

type factoryFunc func(string, string) resources.Client

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration config.Config) Client {
	return NewWithFactory(configuration, resources.NewClientWithBaseURI)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
func NewWithFactory(configuration config.Config, factory factoryFunc) Client {
	return &client{
		config:  configuration,
		factory: factory,
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *client) ForSubscription(subID string) error {
	c.internal = c.factory(subID, azure.PublicCloud.ResourceManagerEndpoint)
	c.internal.RequestInspector = LogRequest()
	c.internal.ResponseInspector = LogResponse()
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Exists returns if the resource ID exists in Azure.
func (c *client) Exists(ctx context.Context, resource string) (bool, error) {
	response, err := c.internal.CheckExistenceByID(ctx, resource)
	if err != nil {
		if response.IsHTTPStatus(http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func LogRequest() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				log.Println(err)
			}
			dump, _ := httputil.DumpRequestOut(r, true)
			log.Println(string(dump))
			return r, err
		})
	}
}

func LogResponse() autorest.RespondDecorator {
	return func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			err := p.Respond(r)
			if err != nil {
				log.Println(err)
			}
			dump, _ := httputil.DumpResponse(r, true)
			log.Println(string(dump))
			return err
		})
	}
}
