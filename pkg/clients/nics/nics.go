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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"

	"github.com/davecgh/go-spew/spew"
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
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	if found {
		if c.Done(ctx, local) {
			return true, nil
		} else {
			spew.Dump("not done")
			return false, nil
		}
	}

	// TODO(ace): use spec pattern from other clients
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
			return false, errors.New("must have at least one IP configuration")
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

	if _, err = c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
		return false, err
	}
	return false, nil
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (network.Interface, error) {
	local, err := c.convert(obj)
	if err != nil {
		return network.Interface{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
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
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, expand)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
		return false, nil
	}
	return found, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.NetworkInterface, remote network.Interface) {
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.InterfacePropertiesFormat != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
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

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.NetworkInterface, error) {
	local, ok := obj.(*azurev1alpha1.NetworkInterface)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
