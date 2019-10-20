/*
Copyright 2019 Alexander Eldeib.
*/

package resourcegroups

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type GroupClient struct {
	*service
}

// NewGroupClient returns a reconciler Azure resource groups.
func NewGroupClient(configuration *config.Config) *GroupClient {
	return &GroupClient{
		service: newGroupService(configuration),
	}
}

// ForSubscription is a noop for service style clients
func (c *GroupClient) ForSubscription(ctx context.Context, obj runtime.Object) error {
	// noop for keyvault secrets and kubeclients
	return nil
}

// Ensure creates or updates a resource group in an idempotent manner.
func (c *GroupClient) Ensure(ctx context.Context, obj runtime.Object) (done bool, err error) {
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
		Name(local.Spec.Name),
		Location(local.Spec.Location),
	)

	remote, err = c.service.CreateOrUpdate(ctx, local, spec.Build())
	setStatus(local, remote)
	return err == nil, err
}

// Delete handles deletion of a resource groups.
func (c *GroupClient) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := convert(obj)
	if err != nil {
		return false, err
	}

	return c.service.Delete(ctx, local, log)
}

// setStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func setStatus(local *azurev1alpha1.ResourceGroup, remote resources.Group) {
	local.Status.ID = remote.ID
	if remote.Properties != nil {
		local.Status.ProvisioningState = remote.Properties.ProvisioningState
	}
}

func convert(obj runtime.Object) (*azurev1alpha1.ResourceGroup, error) {
	local, ok := obj.(*azurev1alpha1.ResourceGroup)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
