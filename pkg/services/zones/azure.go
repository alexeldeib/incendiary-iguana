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

	var zones []string
	res, err := client.ListComplete(ctx)
	if err != nil {
		return zones, err
	}

	for res.NotDone() {
		resSku := res.Value()
		if strings.EqualFold(*resSku.Name, local.Spec.SKU) {
			// Use map for easy deletion and iteration
			availableZones := make(map[string]bool)
			for _, locationInfo := range *resSku.LocationInfo {
				for _, zone := range *locationInfo.Zones {
					availableZones[zone] = true
				}
				if strings.EqualFold(*locationInfo.Location, local.Spec.Location) {
					for _, restriction := range *resSku.Restrictions {
						// Can't deploy anything in this subscription in this location. Bail out.
						if restriction.Type == compute.Location {
							return []string{}, errors.Errorf("rejecting sku: %s in location: %s due to susbcription restriction", local.Spec.SKU, local.Spec.Location)
						}
						// May be able to deploy one or more zones to this location.
						for _, restrictedZone := range *restriction.RestrictionInfo.Zones {
							delete(availableZones, restrictedZone)
						}
					}
					// Back to slice. Empty is fine, and will deploy the VM to some FD/UD (no point in configuring this until supported at higher levels)
					result := make([]string, 0)
					for availableZone := range availableZones {
						result = append(result, availableZone)
					}
					// Lexical sort so comparisons work in tests
					sort.Strings(result)
					zones = result
				}
			}
		}
		err = res.NextWithContext(ctx)
		if err != nil {
			return zones, err
		}
	}

	return zones, nil
}
