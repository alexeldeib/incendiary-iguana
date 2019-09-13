/*
Copyright 2019 Alexander Eldeib.
*/

package vms

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal compute.VirtualMachinesClient
	config   *config.Config
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
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(subID string) error {
	c.internal = c.factory(subID)
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.VM) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, compute.InstanceView)
	found := !remote.HasHTTPStatus(http.StatusNotFound, http.StatusConflict)
	c.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	if found {
		if c.Done(ctx, local) {
			return true, nil
		}
		spew.Dump("not done")
	}

	randomPassword, err := GenerateRandomString(32)
	if err != nil {
		return false, errors.Wrapf(err, "failed to generate random string")
	}

	spec := compute.VirtualMachine{
		Location: &local.Spec.Location,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(local.Spec.SKU),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: to.StringPtr("Canonical"),
					Offer:     to.StringPtr("UbuntuServer"),
					Sku:       to.StringPtr("18.04-LTS"),
					Version:   to.StringPtr("latest"),
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: &local.Spec.PrimaryNIC,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  to.StringPtr(local.Spec.Name),
				AdminUsername: to.StringPtr("azureuser"),
				AdminPassword: to.StringPtr(randomPassword),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: to.BoolPtr(false),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path:    to.StringPtr("/home/azureuser/.ssh/authorized_keys"),
								KeyData: to.StringPtr("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDHu5TlJfCSJ0KkimVWoUbnvFZvhaN7sQYXYTtAqmftY4B8J8PboDHvGWN8ImFCEd1Zld8MIGwE5b43cHfoymd4rmHcNiuGnFpFGyULOqom/eRBy7FEaGCPAq/T/SAJ3G4843wryUOQcr71zoJbbgICYgtxhAINzHp+e/b7t6FujJ9D0G9aYxsEmgcsOGIW6TVVwQ3fbB1BPpWxVATbGnipklve7UbSeu1E0Kci4pzR3ffh6Ihauvni26e3ImgFlzHXLDwD/vZjyFL2/VyXjaF9EKLYu0DMhYbAolgqKRKmYXPq7w/1TPsIiY8DqVuNmwkouO5sR8oecxXVXpNF7BT7 aleldeib@redmond@MININT-LU1PG6L"),
							},
						},
					},
				},
			},
		},
		Zones: &[]string{"2"},
	}
	spew.Dump(spec)
	if _, err = c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
		spew.Dump(err)
		return false, err
	}
	spew.Dump("passed err")
	return false, nil
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.VM) (compute.VirtualMachine, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name, compute.InstanceView)
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.VM) (bool, error) {
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
		return false, nil
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
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.VM) bool {
	return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
}

// InProgress
func (c *Client) InProgress(ctx context.Context, local *azurev1alpha1.VM) bool {
	return local.Status.ProvisioningState != nil
}

// func (c *Client) NeedsUpdate(local *azurev1alpha1.VM, remote compute.VirtualMachine) bool {
// 	if remote.Sku != nil {
// 		if !strings.EqualFold(string(local.Spec.SKU.Name), string(remote.Sku.Name)) {
// 			spew.Dump("changed sku name")
// 			return true
// 		}
// 		if !strings.EqualFold(string(local.Spec.SKU.Tier), string(remote.Sku.Tier)) {
// 			spew.Dump("changed sku tier")
// 			return true
// 		}
// 		if remote.Sku.Capacity != nil && local.Spec.SKU.Capacity != *remote.Sku.Capacity {
// 			spew.Dump("changed capacity")
// 			return true
// 		}
// 	}
// 	if remote.Location != nil && strings.EqualFold(*remote.Location, local.Spec.Location) {
// 		spew.Dump("changed Location")
// 		return true
// 	}
// 	return false
// }

func (c *Client) TryAuthorize(ctx context.Context, obj runtime.Object) error {
	local, ok := obj.(*azurev1alpha1.VM)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.ForSubscription(local.Spec.SubscriptionID)
}

func (c *Client) TryEnsure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.VM)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Ensure(ctx, local)
}

func (c *Client) TryDelete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.VM)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Delete(ctx, local)
}

// https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/60b7c6058550ae694935fb03103460a2efa4e332/pkg/cloud/azure/services/virtualmachines/virtualmachines.go#L215
func GenerateRandomString(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), err
}
