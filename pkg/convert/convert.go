package convert

import (
	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/Azure/azure-sdk-for-go/services/servicebus/mgmt/2017-04-01/servicebus"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

func RedisToCreateParameters(local *azurev1alpha1.Redis) redis.CreateParameters {
	return redis.CreateParameters{
		Location: &local.Spec.Location,
		CreateProperties: &redis.CreateProperties{
			EnableNonSslPort: &local.Spec.EnableNonSslPort,
			Sku: &redis.Sku{
				Name:     redis.SkuName(local.Spec.SKU.Name),
				Family:   redis.SkuFamily(local.Spec.SKU.Family),
				Capacity: &local.Spec.SKU.Capacity,
			},
		},
	}
}

func SBNamespace(local *azurev1alpha1.ServiceBusNamespace) servicebus.SBNamespace {
	return servicebus.SBNamespace{
		Location: &local.Spec.Location,
		Sku: &servicebus.SBSku{
			Name:     servicebus.SkuName(local.Spec.SKU.Name),
			Tier:     servicebus.SkuTier(local.Spec.SKU.Tier),
			Capacity: &local.Spec.SKU.Capacity,
		},
	}
}

func StorageAccountToCreateParameters(local *azurev1alpha1.StorageAccount) storage.AccountCreateParameters {
	return storage.AccountCreateParameters{
		Sku: &storage.Sku{
			Kind: storage.Storage,
		},
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
		Location:                          &local.Spec.Location,
	}
}
