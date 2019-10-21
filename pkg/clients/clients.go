package clients

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/alexeldeib/incendiary-iguana/pkg/authorizer"
)

// NewGroupsClient returns an authenticated client using the provided authorizer factory.
func NewGroupsClient(sub string, factory authorizer.Factory) (resources.GroupsClient, error) {
	client := resources.NewGroupsClient(sub)
	authorizer, err := factory.New()
	if err != nil {
		return resources.GroupsClient{}, err
	}
	client.Authorizer = authorizer
	return client, nil
}

// NewVirtualNetworksClient returns an authenticated client using the provided authorizer factory.
func NewVirtualNetworksClient(sub string, factory authorizer.Factory) (network.VirtualNetworksClient, error) {
	client := network.NewVirtualNetworksClient(sub)
	authorizer, err := factory.New()
	if err != nil {
		return network.VirtualNetworksClient{}, err
	}
	client.Authorizer = authorizer
	return client, nil
}

// NewSubnetsClient returns an authenticated client using the provided authorizer factory.
func NewSubnetsClient(sub string, factory authorizer.Factory) (network.SubnetsClient, error) {
	client := network.NewSubnetsClient(sub)
	authorizer, err := factory.New()
	if err != nil {
		return network.SubnetsClient{}, err
	}
	client.Authorizer = authorizer
	return client, nil
}

// NewResourceSkusClient returns an authenticated client using the provided authorizer factory.
func NewResourceSkusClient(sub string, factory authorizer.Factory) (compute.ResourceSkusClient, error) {
	client := compute.NewResourceSkusClient(sub)
	authorizer, err := factory.New()
	if err != nil {
		return compute.ResourceSkusClient{}, err
	}
	client.Authorizer = authorizer
	return client, nil
}
