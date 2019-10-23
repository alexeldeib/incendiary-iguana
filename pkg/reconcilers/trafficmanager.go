/*
Copyright 2019 Alexander Eldeib.
*/

package reconcilers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/trafficmanager/mgmt/2018-04-01/trafficmanager"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/convert"
	"github.com/alexeldeib/incendiary-iguana/pkg/services"
)

type TrafficManagerReconciler struct {
	Service    *services.TrafficManagerService
	Kubeclient ctrl.Client
}

func (r *TrafficManagerReconciler) Ensure(ctx context.Context, obj runtime.Object) (done bool, err error) {
	local, err := r.convert(obj)
	if err != nil {
		return false, err
	}

	if err := r.Service.CreateOrUpdate(ctx, local, convert.TrafficManagerProfile(local)); err != nil {
		return false, err
	}

	remote, err := r.Service.Get(ctx, local)
	r.SetStatus(ctx, local, remote)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && !remote.HasHTTPStatus(http.StatusNotFound, http.StatusConflict) {
		return found, err
	}

	return r.Done(ctx, local), nil
}

func (r *TrafficManagerReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (done bool, err error) {
	local, err := r.convert(obj)
	if err != nil {
		return false, err
	}
	err = r.Service.Delete(ctx, local, log)
	return err == nil, err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (r *TrafficManagerReconciler) SetStatus(ctx context.Context, local *azurev1alpha1.TrafficManager, remote trafficmanager.Profile) {
	local.Status.ID = remote.ID
	if remote.ProfileProperties != nil {
		local.Status.FQDN = remote.ProfileProperties.DNSConfig.Fqdn
		local.Status.ProfileMonitorStatus = string(remote.ProfileProperties.MonitorConfig.ProfileMonitorStatus)
	}
}

// Done checks the current state of the CRD against the desired end state.
func (r *TrafficManagerReconciler) Done(ctx context.Context, local *azurev1alpha1.TrafficManager) bool {
	// TODO(ace): make this check individual endpoints? what about ICMs?
	return local.Status.ProfileMonitorStatus == "Online"
}

// Get returns a virtual network.
func (r *TrafficManagerReconciler) Get(ctx context.Context, local *azurev1alpha1.TrafficManager) (trafficmanager.Profile, error) {
	return r.Service.Get(ctx, local)
}

// remove?
// GetProfileStatus returns the status of an entire Azure TM.
func (r *TrafficManagerReconciler) GetProfileStatus(ctx context.Context, local *azurev1alpha1.TrafficManager) (string, error) {
	res, err := r.Service.Get(ctx, local)
	if err != nil {
		return "", err
	}
	return string(res.ProfileProperties.MonitorConfig.ProfileMonitorStatus), nil
}

// remove?
// GetEndpointStatus returns the status of one endpoint within an Azure Traffic Manager.
func (r *TrafficManagerReconciler) GetEndpointStatus(ctx context.Context, local *azurev1alpha1.TrafficManager, name string) (string, error) {
	profile, err := r.Service.Get(ctx, local)
	if err != nil {
		return "", err
	}
	for _, ep := range *profile.ProfileProperties.Endpoints {
		if *ep.Name == name {
			return string(ep.EndpointMonitorStatus), nil
		}
	}
	return "", errors.New("endpoint not found in current tm configuration")
}

func (r *TrafficManagerReconciler) convert(obj runtime.Object) (*azurev1alpha1.TrafficManager, error) {
	local, ok := obj.(*azurev1alpha1.TrafficManager)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
