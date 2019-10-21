/*
Copyright 2019 Alexander Eldeib.
*/

package virtualnetworks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/authorizer"
)

const expand string = ""

type VnetReconciler struct {
	*service
}

// NewVnetReconciler returns a reconciler Azure resource groups.
func NewVnetReconciler(credentials authorizer.Factory) *VnetReconciler {
	return &VnetReconciler{
		newService(credentials),
	}
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *VnetReconciler) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := convert(obj)
	if err != nil {
		return false, err
	}
	remote, err := c.service.Get(ctx, local)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	setStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	var spec *Spec
	if found {
		spec = NewSpecWithRemote(&remote)
		if !spec.NeedsUpdate(local) {
			return true, nil
		}
	} else {
		spec = NewSpec()
	}

	spec.Set(
		Name(&local.Spec.Name),
		Location(&local.Spec.Location),
		AddressSpaces(local.Spec.Addresses), // TODO(ace): declarative vs patch for merging over existing fields?
	)

	_, err = c.service.CreateOrUpdate(ctx, local, spec.Build())
	return false, err
}

// Get returns a virtual network.
func (c *VnetReconciler) Get(ctx context.Context, obj runtime.Object) (network.VirtualNetwork, error) {
	local, err := convert(obj)
	if err != nil {
		return network.VirtualNetwork{}, err
	}
	return c.service.Get(ctx, local)
}

// Delete handles deletion of a virtual network.
func (c *VnetReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := convert(obj)
	if err != nil {
		return false, err
	}

	return c.service.Delete(ctx, local, log)
}

// setStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func setStatus(local *azurev1alpha1.VirtualNetwork, remote network.VirtualNetwork) {
	local.Status.ID = remote.ID
	if remote.VirtualNetworkPropertiesFormat != nil {
		local.Status.ProvisioningState = remote.VirtualNetworkPropertiesFormat.ProvisioningState
	}
}

func convert(obj runtime.Object) (*azurev1alpha1.VirtualNetwork, error) {
	local, ok := obj.(*azurev1alpha1.VirtualNetwork)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
