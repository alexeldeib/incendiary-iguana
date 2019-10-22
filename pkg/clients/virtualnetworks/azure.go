/*
Copyright 2019 Alexander Eldeib.
*/

package virtualnetworks

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/authorizer"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
	"github.com/alexeldeib/incendiary-iguana/pkg/constants"
)

type service struct {
	factory authorizer.Factory
}

func newService(factory authorizer.Factory) *service {
	return &service{
		factory,
	}
}

func (s *service) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.VirtualNetwork, remote network.VirtualNetwork) (done bool, err error) {
	client, err := clients.NewVirtualNetworksClient(local.Spec.SubscriptionID, s.factory)
	if err != nil {
		return false, err
	}

	var future network.VirtualNetworksCreateOrUpdateFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = network.VirtualNetworksCreateOrUpdateFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		createFuture, err := client.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, remote)
		if err != nil {
			return false, err
		}
		future = createFuture
		local.Status.Future = &[]byte{}
	}

	log.Info("checking deletion status")
	done, err = future.DoneWithContext(ctx, &client)
	if err != nil {
		if res := future.Response(); res != nil && res.StatusCode == http.StatusNotFound {
			// Not found is successful delete on a resource.
			return false, nil
		}
		return false, err
	}

	if done {
		local.Status.Future = nil
	} else {
		*local.Status.Future, err = future.MarshalJSON()
	}

	return done, err
}

func (s *service) Get(ctx context.Context, local *azurev1alpha1.VirtualNetwork) (result network.VirtualNetwork, err error) {
	client, err := clients.NewVirtualNetworksClient(local.Spec.SubscriptionID, s.factory)
	if err != nil {
		return network.VirtualNetwork{}, err
	}
	return client.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, constants.Empty)
}

func (s *service) Delete(ctx context.Context, local *azurev1alpha1.VirtualNetwork, log logr.Logger) (bool, error) {
	client, err := clients.NewVirtualNetworksClient(local.Spec.SubscriptionID, s.factory)
	if err != nil {
		return false, err
	}

	var future network.VirtualNetworksDeleteFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = network.VirtualNetworksDeleteFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		deleteFuture, err := client.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
		if err != nil {
			return false, err
		}
		future = deleteFuture
		local.Status.Future = &[]byte{}
	}

	log.Info("checking deletion status")
	done, err := future.DoneWithContext(ctx, &client)
	if err != nil {
		if res := future.Response(); res != nil && res.StatusCode == http.StatusNotFound {
			// Not found is successful delete on a resource.
			return false, nil
		}
		return false, err
	}

	if done {
		local.Status.Future = nil
	} else {
		*local.Status.Future, err = future.MarshalJSON()
	}

	return !done, err
}
