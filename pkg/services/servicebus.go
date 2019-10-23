/*
Copyright 2019 Alexander Eldeib.
*/

package services

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
	"github.com/alexeldeib/incendiary-iguana/pkg/convert"
)

type ServiceBusService struct {
	autorest.Authorizer
}

func (s *ServiceBusService) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace, log logr.Logger) (bool, error) {
	client, err := clients.NewServiceBusNamespacesClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future servicebus.NamespacesCreateOrUpdateFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = servicebus.NamespacesCreateOrUpdateFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning creation")
		future, err = client.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, convert.SBNamespace(local))
		if err != nil {
			return false, nil
		}
		local.Status.Future = &[]byte{}
	}

	log.Info("checking creation status")
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

func (s *ServiceBusService) Get(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) (servicebus.SBNamespace, error) {
	client, err := clients.NewServiceBusNamespacesClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return servicebus.SBNamespace{}, err
	}
	return client.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

func (s *ServiceBusService) ListKeys(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) (servicebus.AccessKeys, error) {
	client, err := clients.NewServiceBusNamespacesClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return servicebus.AccessKeys{}, err
	}
	return client.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name, "RootManageSharedAccessKey")
}

func (s *ServiceBusService) Delete(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace, log logr.Logger) (bool, error) {
	client, err := clients.NewServiceBusNamespacesClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future servicebus.NamespacesDeleteFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = servicebus.NamespacesDeleteFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		future, err = client.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
		if err != nil {
			return false, err
		}
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
