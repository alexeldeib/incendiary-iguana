package convert

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/trafficmanager/mgmt/trafficmanager"
	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/Azure/azure-sdk-for-go/services/servicebus/mgmt/2017-04-01/servicebus"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

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

func TrafficManagerProfile(local *azurev1alpha1.TrafficManager) trafficmanager.Profile {
	spec := trafficmanager.Profile{
		ProfileProperties: &trafficmanager.ProfileProperties{
			ProfileStatus:        trafficmanager.ProfileStatus(local.Spec.ProfileStatus),
			TrafficRoutingMethod: trafficmanager.TrafficRoutingMethod(local.Spec.TrafficRoutingMethod),
			MonitorConfig: &trafficmanager.MonitorConfig{
				Protocol:                  trafficmanager.MonitorProtocol(local.Spec.MonitorConfig.Protocol),
				Port:                      local.Spec.MonitorConfig.Port,
				Path:                      to.StringPtr("/healthz"),
				IntervalInSeconds:         local.Spec.MonitorConfig.IntervalInSeconds,
				TimeoutInSeconds:          local.Spec.MonitorConfig.TimeoutInSeconds,
				ToleratedNumberOfFailures: local.Spec.MonitorConfig.ToleratedNumberOfFailures,
				CustomHeaders:             &[]trafficmanager.MonitorConfigCustomHeadersItem{},
				ExpectedStatusCodeRanges:  &[]trafficmanager.MonitorConfigExpectedStatusCodeRangesItem{},
			},
			Endpoints: &[]trafficmanager.Endpoint{},
			DNSConfig: &trafficmanager.DNSConfig{
				RelativeName: local.Spec.DNSConfig.RelativeName,
				Fqdn:         to.StringPtr(fmt.Sprintf("%s.trafficmanager.net", *local.Spec.DNSConfig.RelativeName)),
				TTL:          local.Spec.DNSConfig.TTL,
			},
		},
		Location: to.StringPtr("global"),
	}

	if local.Spec.MonitorConfig.Path != nil {
		spec.ProfileProperties.MonitorConfig.Path = local.Spec.MonitorConfig.Path
	}

	if local.Spec.MonitorConfig.CustomHeaders != nil {
		for _, header := range *local.Spec.MonitorConfig.CustomHeaders {
			new := trafficmanager.MonitorConfigCustomHeadersItem{
				Name:  header.Name,
				Value: header.Value,
			}
			*spec.ProfileProperties.MonitorConfig.CustomHeaders = append(*spec.ProfileProperties.MonitorConfig.CustomHeaders, new)
		}
	}

	if local.Spec.MonitorConfig.CustomHeaders != nil {
		for _, spread := range *local.Spec.MonitorConfig.ExpectedStatusCodeRanges {
			new := trafficmanager.MonitorConfigExpectedStatusCodeRangesItem{
				Min: spread.Min,
				Max: spread.Max,
			}
			*spec.ProfileProperties.MonitorConfig.ExpectedStatusCodeRanges = append(*spec.ProfileProperties.MonitorConfig.ExpectedStatusCodeRanges, new)
		}
	}

	if local.Spec.Endpoints != nil {
		for _, ep := range *local.Spec.Endpoints {
			endpointSpec := trafficmanager.Endpoint{
				Name: to.StringPtr(ep.Name),
				Type: to.StringPtr("Microsoft.Network/trafficManagerProfiles/externalEndpoints"),
				EndpointProperties: &trafficmanager.EndpointProperties{
					Target:           ep.Properties.Target,
					Weight:           ep.Properties.Weight,
					Priority:         to.Int64Ptr(ep.Properties.Priority),
					EndpointLocation: to.StringPtr(ep.Properties.EndpointLocation),
				},
			}
			if ep.Properties.CustomHeaders != nil {
				endpointSpec.CustomHeaders = &[]trafficmanager.EndpointPropertiesCustomHeadersItem{}
				for _, header := range *ep.Properties.CustomHeaders {
					item := trafficmanager.EndpointPropertiesCustomHeadersItem{
						Name:  header.Name,
						Value: header.Value,
					}
					*endpointSpec.CustomHeaders = append(*endpointSpec.CustomHeaders, item)
				}
			}
			*spec.ProfileProperties.Endpoints = append(*spec.ProfileProperties.Endpoints, endpointSpec)
		}
	}
	return spec
}

func ResourceGroup(local *azurev1alpha1.ResourceGroup) resources.Group {
	return resources.Group{
		Location: &local.Spec.Location,
	}
}
