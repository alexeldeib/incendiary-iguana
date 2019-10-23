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
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/convert"
)

type ResourceGroupReconciler struct {
	Service    *ResourceGroupService
	Kubeclient ctrl.Client
	Scheme     *runtime.Scheme
}

// Ensure creates or updates a redis cache in an idempotent manner.
func (r *ResourceGroupReconciler) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, err := r.convert(obj)
	if err != nil {
		return false, err
	}

	remote, err := r.Service.Get(ctx, local)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	r.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	_, err = r.Service.CreateOrUpdate(ctx, local, convert.ResourceGroup(local))
	return err == nil, err
}

// Delete handles deletion of a resource groups.
func (r *ResourceGroupReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := r.convert(obj)
	if err != nil {
		return false, err
	}
	return r.Service.Delete(ctx, local, log)
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (r *ResourceGroupReconciler) SetStatus(local *azurev1alpha1.ResourceGroup, remote resources.Group) {
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.Properties != nil {
		local.Status.ProvisioningState = remote.Properties.ProvisioningState
	}
}

func (r *ResourceGroupReconciler) convert(obj runtime.Object) (*azurev1alpha1.ResourceGroup, error) {
	local, ok := obj.(*azurev1alpha1.ResourceGroup)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
