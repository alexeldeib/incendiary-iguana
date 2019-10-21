/*
Copyright 2019 Alexander Eldeib.
*/

package zones

import (
	"context"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/pkg/errors"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/authorizer"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
)

type Service struct {
	factory authorizer.Factory
}

func NewZoneService(factory authorizer.Factory) *Service {
	return &Service{
		factory,
	}
}

func (s *Service) Get(ctx context.Context, local *azurev1alpha1.VM) ([]string, error) {
	client, err := clients.NewResourceSkusClient(local.Spec.SubscriptionID, s.factory)
	if err != nil {
		return nil, err
	}

	res, err := client.ListComplete(ctx)
	if err != nil {
		return nil, err
	}

	var skus []compute.ResourceSku
	for res.NotDone() {
		skus = append(skus, res.Value())
		err = res.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return filterZones(skus, local.Spec.SubscriptionID, local.Spec.Location, local.Spec.SKU)
}

func filterZones(skus []compute.ResourceSku, sub, location, sku string) ([]string, error) {
	for _, val := range skus {
		if strings.EqualFold(*val.Name, sku) {
			for _, locationInfo := range *val.LocationInfo {
				if strings.EqualFold(*locationInfo.Location, location) {
					available := make(map[string]bool)
					for _, zone := range *locationInfo.Zones {
						available[zone] = true
					}
					for _, restriction := range *val.Restrictions {
						// Can't deploy anything in this subscription in this location. Bail out.
						if restriction.Type == compute.Location {
							return nil, errors.Errorf("rejecting sku: %s in location: %s due to susbcription restriction", sku, location)
						}
						// May be able to deploy one or more zones to this location.
						for _, restricted := range *restriction.RestrictionInfo.Zones {
							delete(available, restricted)
						}
					}
					// Back to slice. Empty is fine, and will deploy the VM to some FD/UD (no point in configuring this until supported at higher levels)
					result := make([]string, 0)
					for az := range available {
						result = append(result, az)
					}
					// Lexical sort so comparisons work in tests
					sort.Strings(result)
					return result, nil
				}
			}
		}
	}
	return nil, nil
}
