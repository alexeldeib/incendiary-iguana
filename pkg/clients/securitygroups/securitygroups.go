/*
Copyright 2019 Alexander Eldeib.
*/

package securitygroups

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const expand string = ""

type Client struct {
	factory  factoryFunc
	internal network.SecurityGroupsClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) network.SecurityGroupsClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, network.NewSecurityGroupsClient)
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
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.SecurityGroup) error {
	spec := network.SecurityGroup{
		Location: &local.Spec.Location,
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &[]network.SecurityRule{},
		},
	}
	for _, rule := range local.Spec.Rules {
		newRule := network.SecurityRule{
			Name: &rule.Name,
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Protocol:                 rule.Protocol,
				SourceAddressPrefix:      rule.SourceAddressPrefix,
				SourcePortRange:          rule.SourcePortRange,
				DestinationAddressPrefix: rule.DestinationAddressPrefix,
				DestinationPortRange:     rule.DestinationPortRange,
				Access:                   rule.Access,
				Direction:                rule.Direction,
				Priority:                 rule.Priority,
			},
		}
		*spec.SecurityGroupPropertiesFormat.SecurityRules = append(*spec.SecurityGroupPropertiesFormat.SecurityRules, newRule)
	}
	_, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec)
	return err
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.SecurityGroup) (network.SecurityGroup, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.SecurityGroup) error {
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return err
		}
	}
	return nil
}
