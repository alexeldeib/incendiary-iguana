/*
Copyright 2019 Alexander Eldeib.
*/

package subnets

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/authorizer"
	"github.com/alexeldeib/incendiary-iguana/pkg/constants"
)

type SubnetReconciler struct {
	*service
}

// NewSubnetReconciler returns a reconciler Azure resource groups.
func NewSubnetReconciler(credentials authorizer.Factory) *SubnetReconciler {
	return &SubnetReconciler{
		newService(credentials),
	}
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *SubnetReconciler) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := convert(obj)
	if err != nil {
		return false, err
	}

	remote, err := c.service.Get(ctx, local)
	setStatus(local, remote)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && found {
		return found, err
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
		Name(local.Spec.Name),
		Address(local.Spec.Subnet),
	)

	return c.service.CreateOrUpdate(ctx, local, spec.Build())
}

// Get returns a virtual network.
func (c *SubnetReconciler) Get(ctx context.Context, obj runtime.Object) (network.Subnet, error) {
	local, err := convert(obj)
	if err != nil {
		return network.Subnet{}, err
	}
	return c.service.Get(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name, constants.Empty)
}

// Delete handles deletion of a virtual network.
func (c *SubnetReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := convert(obj)
	if err != nil {
		return false, err
	}
	future, err := c.service.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name)
	if err != nil {
		// Not found is a successful delete
		if resp := future.Response(); resp != nil && resp.StatusCode != http.StatusNotFound {
			return false, err
		}
	}
	remote, err := c.service.Get(ctx, local.Spec.ResourceGroup, local.Spec.Network, local.Spec.Name, constants.Empty)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	setStatus(local, remote)
	if err != nil && remote.IsHTTPStatus(http.StatusNotFound) {
		return false, nil
	}
	return found, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func setStatus(local *azurev1alpha1.Subnet, remote network.Subnet) {
	local.Status.ID = remote.ID
	if remote.SubnetPropertiesFormat != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
}

func convert(obj runtime.Object) (*azurev1alpha1.Subnet, error) {
	local, ok := obj.(*azurev1alpha1.Subnet)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
