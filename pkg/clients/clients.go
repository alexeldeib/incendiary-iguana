package clients

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
	kvsecret "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/servicebus/mgmt/2017-04-01/servicebus"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-sdk-for-go/services/trafficmanager/mgmt/2018-04-01/trafficmanager"
	"github.com/Azure/go-autorest/autorest"
)

// NewAccountsClient returns an authenticated client using the provided authorizer factory.
func NewAccountsClient(sub string, authorizer autorest.Authorizer) (storage.AccountsClient, error) {
	client := storage.NewAccountsClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewGroupsClient returns an authenticated client using the provided authorizer factory.
func NewGroupsClient(sub string, authorizer autorest.Authorizer) (resources.GroupsClient, error) {
	client := resources.NewGroupsClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewInterfacesClient returns an authenticated client using the provided authorizer factory.
func NewInterfacesClient(sub string, authorizer autorest.Authorizer) (network.InterfacesClient, error) {
	client := network.NewInterfacesClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewVaultsClient returns an authenticated client.
func NewVaultsClient(sub string, authorizer autorest.Authorizer) (keyvault.VaultsClient, error) {
	client := keyvault.NewVaultsClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewSecretsClient returns an authenticated client.
func NewSecretsClient(authorizer autorest.Authorizer) kvsecret.BaseClient {
	kvclient := kvsecret.New()
	kvclient.Authorizer = authorizer
	return kvclient
}

// NewLoadBalancersClient returns an authenticated client using the provided authorizer factory.
func NewLoadBalancersClient(sub string, authorizer autorest.Authorizer) (network.LoadBalancersClient, error) {
	client := network.NewLoadBalancersClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewPublicIPAddressesClient returns an authenticated client using the provided authorizer factory.
func NewPublicIPAddressesClient(sub string, authorizer autorest.Authorizer) (network.PublicIPAddressesClient, error) {
	client := network.NewPublicIPAddressesClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewProfilesClient returns an authenticated client using the provided authorizer factory.
func NewProfilesClient(sub string, authorizer autorest.Authorizer) (trafficmanager.ProfilesClient, error) {
	client := trafficmanager.NewProfilesClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewRedisClient returns an authenticated client using the provided authorizer factory.
func NewRedisClient(sub string, authorizer autorest.Authorizer) (redis.Client, error) {
	client := redis.NewClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewResourceSkusClient returns an authenticated client using the provided authorizer factory.
func NewResourceSkusClient(sub string, authorizer autorest.Authorizer) (compute.ResourceSkusClient, error) {
	client := compute.NewResourceSkusClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewSecurityGroupsClient returns an authenticated client using the provided authorizer factory.
func NewSecurityGroupsClient(sub string, authorizer autorest.Authorizer) (network.SecurityGroupsClient, error) {
	client := network.NewSecurityGroupsClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewServiceBusNamespacesClient returns an authenticated client using the provided authorizer factory.
func NewServiceBusNamespacesClient(sub string, authorizer autorest.Authorizer) (servicebus.NamespacesClient, error) {
	client := servicebus.NewNamespacesClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewSubnetsClient returns an authenticated client using the provided authorizer factory.
func NewSubnetsClient(sub string, authorizer autorest.Authorizer) (network.SubnetsClient, error) {
	client := network.NewSubnetsClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewVirtualMachinesClient returns an authenticated client using the provided authorizer factory.
func NewVirtualMachinesClient(sub string, authorizer autorest.Authorizer) (compute.VirtualMachinesClient, error) {
	client := compute.NewVirtualMachinesClient(sub)
	client.Authorizer = authorizer
	return client, nil
}

// NewVirtualNetworksClient returns an authenticated client using the provided authorizer factory.
func NewVirtualNetworksClient(sub string, authorizer autorest.Authorizer) (network.VirtualNetworksClient, error) {
	client := network.NewVirtualNetworksClient(sub)
	client.Authorizer = authorizer
	return client, nil
}
