/*
Copyright 2019 Alexander Eldeib.
*/

package sql

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2015-05-01-preview/sql"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
	"github.com/go-logr/logr"
)

type SQLServerService struct {
	autorest.Authorizer
}

func (s *SQLServerService) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.AzureSqlServer, log logr.Logger) (bool, error) {
	client, err := clients.NewSQLServersClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future sql.ServersCreateOrUpdateFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = sql.ServersCreateOrUpdateFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		future, err = client.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.ObjectMeta.Name, SQLServerToCreateParameters(local))
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

func (s *SQLServerService) Get(ctx context.Context, local *azurev1alpha1.AzureSqlServer) (sql.Server, error) {
	client, err := clients.NewSQLServersClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return sql.Server{}, err
	}
	return client.Get(ctx, local.Spec.ResourceGroup, local.ObjectMeta.Name)
}

// func (s *SQLServerService) ListKeys(ctx context.Context, local *azurev1alpha1.SQLServer) (sql.AccessKeys, error) {
// 	client, err := clients.NewSQLServersClient(local.Spec.SubscriptionID, s.Authorizer)
// 	if err != nil {
// 		return sql.AccessKeys{}, err
// 	}
// 	return client.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name)
// }

func (s *SQLServerService) Delete(ctx context.Context, local *azurev1alpha1.AzureSqlServer, log logr.Logger) (bool, error) {
	client, err := clients.NewSQLServersClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future sql.ServersDeleteFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = sql.ServersDeleteFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		deleteFuture, err := client.Delete(ctx, local.Spec.ResourceGroup, local.ObjectMeta.Name)
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
