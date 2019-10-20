/*
Copyright 2019 Alexander Eldeib.
*/

package vms

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/disks"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/zones"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal compute.VirtualMachinesClient
	config   *config.Config
	disks    *disks.Client
	zones    *zones.Client
}

type factoryFunc func(subscriptionID string) compute.VirtualMachinesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, compute.NewVirtualMachinesClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
func NewWithFactory(configuration *config.Config, factory factoryFunc) *Client {
	return &Client{
		config:  configuration,
		factory: factory,
		disks:   disks.New(configuration),
		zones:   zones.New(configuration),
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	c.internal = c.factory(local.Spec.SubscriptionID)
	return c.config.AuthorizeClientFromArgs(&c.internal.Client)
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, compute.InstanceView)
	found := !remote.HasHTTPStatus(http.StatusNotFound, http.StatusConflict)
	c.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		if c.Done(ctx, local) {
			if !spec.NeedsUpdate(local) {
				return true, nil
			}
		} else {
			spew.Dump("not done")
			return false, nil
		}
	} else {
		spec = NewSpec()
	}

	zoneFn := func(*Spec) {} // noop unless we set it. default behavior is max fd/ud spread.
	if local.Spec.Zone != nil {
		zoneFn = Zone(*local.Spec.Zone)
	} else if spec.Zone() == nil {
		choices, err := c.zones.Get(ctx, local)
		if err != nil {
			return false, err
		}
		if len(choices) > 0 {
			zoneFn = Zone(choices[rand.Intn(len(choices))])
		}
	}

	spec.Set(
		Name(local.Spec.Name),
		Location(local.Spec.Location),
		Hostname(local.Spec.Name),
		SKU(local.Spec.SKU),
		NICs(local.Spec.PrimaryNIC, local.Spec.SecondaryNICs),
		zoneFn,
	)

	_, err = c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec.Build())
	return false, err
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (compute.VirtualMachine, error) {
	local, err := c.convert(obj)
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, compute.InstanceView)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := c.convert(obj)
	if err != nil {
		return false, err
	}
	future, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, compute.InstanceView)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	c.SetStatus(local, remote)
	if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
		return c.disks.Delete(ctx, local, log)
	}
	return found, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(local *azurev1alpha1.VM, remote compute.VirtualMachine) {
	// Care about 400 and 5xx, not 404.
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.VirtualMachineProperties != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
	local.Status.Zone = nil
	if remote.Zones != nil && len(*remote.Zones) > 0 {
		local.Status.Zone = &((*remote.Zones)[0])
	}
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.VM) bool {
	return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
}

// InProgress
func (c *Client) InProgress(ctx context.Context, local *azurev1alpha1.VM) bool {
	return local.Status.ProvisioningState != nil
}

func (c *Client) NeedsUpdate(local *azurev1alpha1.VM, remote compute.VirtualMachine) bool {
	if !strings.EqualFold(string(local.Spec.SKU), string(remote.VirtualMachineProperties.HardwareProfile.VMSize)) {
		spew.Dump("changed sku name")
		return true
	}
	if !strings.EqualFold(*remote.Location, local.Spec.Location) {
		spew.Dump("changed Location")
		return true
	}
	return false
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.VM, error) {
	local, ok := obj.(*azurev1alpha1.VM)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
