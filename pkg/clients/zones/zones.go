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
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal compute.ResourceSkusClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) compute.ResourceSkusClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, compute.NewResourceSkusClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
func NewWithFactory(configuration *config.Config, factory factoryFunc) *Client {
	return &Client{
		config:  configuration,
		factory: factory,
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(subID string) error {
	c.internal = c.factory(subID)
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Get returns a resource group.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.VM) ([]string, error) {
	var zones []string
	res, err := c.internal.ListComplete(ctx)
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
			return zones, errors.Wrap(err, "could not iterate availability zones")
		}
	}

	return zones, nil
}

// // Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
// func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.VM) (bool, error) {
// 	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
// 	found := !remote.HasHTTPStatus(http.StatusNotFound, http.StatusConflict)
// 	c.SetStatus(local, remote)
// 	if err != nil && found {
// 		return false, err
// 	}

// 	if found {
// 		if c.Done(ctx, local) {
// 			if !c.NeedsUpdate(local, remote) {
// 				return true, nil
// 			}
// 		}
// 		spew.Dump("not done")
// 	}

// 	// Name:         to.StringPtr(fmt.Sprintf("%s_%s_%s_osdisk", local.Spec.SubscriptionID, local.Spec.ResourceGroup, local.Spec.Name)),

// 	if _, err = c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
// 		spew.Dump(err)
// 		return false, err
// 	}
// 	spew.Dump("passed err")
// 	return false, nil
// }

// // SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
// func (c *Client) SetStatus(local *azurev1alpha1.Disk, remote compute.Disk) {
// 	// Care about 400 and 5xx, not 404.
// 	local.Status.ID = remote.ID
// 	local.Status.ProvisioningState = nil
// 	if remote.DiskProperties != nil {
// 		local.Status.ProvisioningState = remote.ProvisioningState
// 	}
// }

// func (c *Client) NeedsUpdate(local *azurev1alpha1.VM, remote compute.Disk) bool {
// if !strings.EqualFold(string(local.Spec.SKU), string(remote.VirtualMachineProperties.HardwareProfile.VMSize)) {
// 	spew.Dump("changed sku name")
// 	return true
// }
// if !strings.EqualFold(*remote.Location, local.Spec.Location) {
// 	spew.Dump("changed Location")
// 	return true
// }
// return false
// }

// func (c *Client) TryAuthorize(ctx context.Context, obj runtime.Object) error {
// 	local, ok := obj.(*azurev1alpha1.VM)
// 	if !ok {
// 		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
// 	}
// 	return c.ForSubscription(local.Spec.SubscriptionID)
// }

// func (c *Client) TryEnsure(ctx context.Context, obj runtime.Object) (bool, error) {
// 	local, ok := obj.(*azurev1alpha1.VM)
// 	if !ok {
// 		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
// 	}
// 	return c.Ensure(ctx, local)
// }

// func (c *Client) TryDelete(ctx context.Context, obj runtime.Object) (bool, error) {
// 	local, ok := obj.(*azurev1alpha1.VM)
// 	if !ok {
// 		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
// 	}
// 	return c.Delete(ctx, local)
// }
