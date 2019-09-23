/*
Copyright 2019 Alexander Eldeib.
*/

package storageaccounts

// import (
// 	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
// 	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

// 	"github.com/alexeldeib/incendiary-iguana/pkg/config"
// )

// type Client struct {
// 	factory factoryFunc
// 	storage.AccountsClient
// }

// type factoryFunc func(subscriptionID string) storage.AccountsClient

// // New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
// func New(configuration *config.Config, kubeclient *ctrl.Client, scheme *runtime.Scheme) *Client {
// 	return NewWithFactory(configuration, kubeclient, servicebus.NewNamespacesClient, scheme)
// }

// // NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// // It uses the factory argument to instantiate new clients for a specific subscription.
// // This can be used to stub Azure client for testing.
// func NewWithFactory(configuration *config.Config, kubeclient *ctrl.Client, factory factoryFunc, scheme *runtime.Scheme) *Client {
// 	return &Client{
// 		config:     configuration,
// 		factory:    factory,
// 		kubeclient: kubeclient,
// 		scheme:     scheme,
// 	}
// }

// // ForSubscription authorizes the client for a given subscription
// func (c *Client) ForSubscription(subID string) error {
// 	c.internal = c.factory(subID)
// 	return c.config.AuthorizeClient(&c.internal.Client)
// }
