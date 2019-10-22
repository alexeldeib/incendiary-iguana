/*
Copyright 2019 Alexander Eldeib.
*/

package services

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
	"github.com/alexeldeib/incendiary-iguana/pkg/convert"
)

type RedisService struct {
	autorest.Authorizer
}

func (s *RedisService) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.Redis, log logr.Logger) (bool, error) {
	client, err := clients.NewRedisClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future redis.CreateFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = redis.CreateFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		future, err = client.Create(ctx, local.Spec.ResourceGroup, local.Spec.Name, convert.RedisToCreateParameters(local))
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

func (s *RedisService) Get(ctx context.Context, local *azurev1alpha1.Redis) (redis.ResourceType, error) {
	client, err := clients.NewRedisClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return redis.ResourceType{}, err
	}
	return client.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

func (s *RedisService) ListKeys(ctx context.Context, local *azurev1alpha1.Redis) (redis.AccessKeys, error) {
	client, err := clients.NewRedisClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return redis.AccessKeys{}, err
	}
	return client.ListKeys(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

func (s *RedisService) Delete(ctx context.Context, local *azurev1alpha1.Redis, log logr.Logger) (bool, error) {
	client, err := clients.NewRedisClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future redis.DeleteFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(*local.Status.Future); err != nil {
			return false, err
		}
		future = redis.DeleteFuture{
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
