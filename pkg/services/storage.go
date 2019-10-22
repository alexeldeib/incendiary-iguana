/*
Copyright 2019 Alexander Eldeib.
*/

package services

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
	"github.com/alexeldeib/incendiary-iguana/pkg/convert"
)

type StorageService struct {
	autorest.Authorizer
}

func (s *StorageService) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.StorageAccount, log logr.Logger) (bool, error) {
	client, err := clients.NewAccountsClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future storage.AccountsCreateFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = storage.AccountsCreateFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		future, err = client.Create(ctx, local.Spec.ResourceGroup, local.Spec.Name, convert.StorageAccountToCreateParameters(local))
		if err != nil {
			return false, nil
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

func (s *StorageService) Get(ctx context.Context, local *azurev1alpha1.StorageAccount) (storage.Account, error) {
	client, err := clients.NewAccountsClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return storage.Account{}, err
	}
	return client.GetProperties(ctx, local.Spec.ResourceGroup, local.Spec.Name, "")
}

func (s *StorageService) ListKeys(ctx context.Context, local *azurev1alpha1.StorageAccount) (storage.AccountListKeysResult, error) {
	client, err := clients.NewAccountsClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return storage.AccountListKeysResult{}, err
	}
	return client.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

func (s *StorageService) Delete(ctx context.Context, local *azurev1alpha1.StorageAccount, log logr.Logger) (autorest.Response, error) {
	client, err := clients.NewAccountsClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return autorest.Response{}, err
	}
	return client.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}
