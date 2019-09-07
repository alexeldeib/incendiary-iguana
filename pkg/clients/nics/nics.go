/*
Copyright 2019 Alexander Eldeib.
*/

package nics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const expand string = ""

type Client struct {
	factory  factoryFunc
	internal network.InterfacesClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) network.InterfacesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, network.NewInterfacesClient)
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
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.NetworkInterface) error {
	spec := network.Interface{
		Location: &local.Spec.Location,
	}
	baseTemplate := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", local.Spec.SubscriptionID, local.Spec.ResourceGroup)
	subnet := fmt.Sprintf("%s/providers/Microsoft.Network/virtualnetworks/%s/subnets/%s", baseTemplate, local.Spec.Network, local.Spec.Subnet)
	ipConfigs := []network.InterfaceIPConfiguration{
		{
			Name: to.StringPtr("ipConfig0"),
			InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
				Primary: to.BoolPtr(true),
				Subnet: &network.Subnet{
					ID: &subnet,
				},
				PrivateIPAllocationMethod: network.Dynamic,
			},
		},
	}
	if local.Spec.IPConfigurations != nil {
		if len(*local.Spec.IPConfigurations) < 1 {
			return errors.New("must have at least one IP configuration")
		}
		// Clear
		ipConfigs = []network.InterfaceIPConfiguration{}
		for index, config := range *local.Spec.IPConfigurations {
			ipConfigName := fmt.Sprintf("ipConfig%s", strconv.Itoa(index))
			innerSpec := buildIPConfig(ipConfigName, subnet, baseTemplate, config)
			ipConfigs = append(ipConfigs, innerSpec)
		}
		ipConfigs[0].Primary = to.BoolPtr(true)
		spec.InterfacePropertiesFormat = &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigs,
		}
	}

	if _, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
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

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.NetworkInterface) (network.Interface, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.NetworkInterface) (bool, error) {
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	return c.SetStatus(ctx, local)
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(ctx context.Context, local *azurev1alpha1.NetworkInterface) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
	// Care about 400 and 5xx, not 404.
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && found {
		if remote.IsHTTPStatus(http.StatusConflict) {
			return found, nil
		}
		return found, err
	}

	local.Status.ID = remote.ID
	local.Status.ProvisioningState = remote.ProvisioningState
	return found, nil
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.NetworkInterface) bool {
	if local.Status.ProvisioningState == nil || *local.Status.ProvisioningState != "Succeeded" {
		return false
	}
	return true
}

func buildIPConfig(name, subnet, baseTemplate string, interfaceConfig azurev1alpha1.InterfaceIPConfig) network.InterfaceIPConfiguration {
	innerSpec := network.InterfaceIPConfiguration{
		Name: &name,
		InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
			Subnet: &network.Subnet{
				ID: &subnet,
			},
			PrivateIPAllocationMethod: network.Dynamic,
		},
	}
	if interfaceConfig.PublicIP != nil {
		ip := fmt.Sprintf("%s/providers/Microsoft.Network/publicIPAddresses/%s", baseTemplate, interfaceConfig.PublicIP.Name)
		innerSpec.InterfaceIPConfigurationPropertiesFormat.PublicIPAddress = &network.PublicIPAddress{
			ID: &ip,
		}
	}
	if interfaceConfig.PrivateIP != nil {
		innerSpec.InterfaceIPConfigurationPropertiesFormat.PrivateIPAllocationMethod = network.IPAllocationMethod("static")
		innerSpec.InterfaceIPConfigurationPropertiesFormat.PrivateIPAddress = interfaceConfig.PrivateIP
	}
	if interfaceConfig.LoadBalancers != nil {
		backendPools := []network.BackendAddressPool{}
		for _, lbConfig := range *interfaceConfig.LoadBalancers {
			poolID := fmt.Sprintf(
				"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.LoadBalancers/%s/backendAddressPools/%s",
				lbConfig.SubscriptionID,
				lbConfig.ResourceGroup,
				lbConfig.LoadBalancer,
				lbConfig.Name,
			)
			pool := network.BackendAddressPool{
				ID: &poolID,
			}
			backendPools = append(backendPools, pool)
		}
		innerSpec.InterfaceIPConfigurationPropertiesFormat.LoadBalancerBackendAddressPools = &backendPools
	}
	return innerSpec
}
