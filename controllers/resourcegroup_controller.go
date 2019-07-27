/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const (
	finalizerName             string = "resourcegroup.azure.alexeldeib.xyz"
	provisioningStateDeleting string = "Deleting"
	provisioningStateNotFound string = "NotFound"
)

// ResourceGroupReconciler reconciles a ResourceGroup object
type ResourceGroupReconciler struct {
	client.Client
	config.Config
	Log          logr.Logger
	GroupsClient resourcegroups.Client
}

// Reconcile reconciles a user request for a Resource Group against Azure.
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups/status,verbs=get;update;patch
func (r *ResourceGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("resourcegroup", req.NamespacedName)

	// Fetch object from Kubernetes API server
	var resourceGroup azurev1alpha1.ResourceGroup
	if err := r.Get(ctx, req.NamespacedName, &resourceGroup); err != nil {
		log.Error(err, "unable to fetch ResourceGroup")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Authorize client
	err := r.GroupsClient.ForSubscription(resourceGroup.Spec.SubscriptionID)
	if err != nil {
		// Don't requeue if fail to instantiate Azure client.
		return ctrl.Result{Requeue: false}, err
	}

	// Fetch state of world
	group, err := r.GroupsClient.Get(ctx, &resourceGroup)
	if err != nil && !group.IsHTTPStatus(http.StatusNotFound) {
		return ctrl.Result{}, err
	}

	// Update our awareness of the state
	oldGeneration := resourceGroup.Status.Generation
	resourceGroup.Status.Generation = resourceGroup.ObjectMeta.GetGeneration()
	if group.Response.IsHTTPStatus(http.StatusNotFound) {
		resourceGroup.Status.ProvisioningState = provisioningStateNotFound
	}
	if group.Properties != nil {
		resourceGroup.Status.ProvisioningState = *group.Properties.ProvisioningState
	}
	if err := r.Status().Update(ctx, &resourceGroup); err != nil {
		log.Info("failed to update status after reconcile")
		return ctrl.Result{}, err
	}

	// Handle deletion/finalizer
	if resourceGroup.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(resourceGroup.ObjectMeta.Finalizers, finalizerName) {
			resourceGroup.ObjectMeta.Finalizers = append(resourceGroup.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &resourceGroup); err != nil {
				log.Info("failed to add finalizer")
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(resourceGroup.ObjectMeta.Finalizers, finalizerName) {
			// our finalizer is present, so lets handle any external dependency
			// We either have 404 or 200. If 200, we may need to begin deletion, wait for deletion to finish,
			if group.IsHTTPStatus(http.StatusOK) {
				var resultErr *multierror.Error
				// Deletion started; wait for completion.
				if *group.Properties.ProvisioningState == provisioningStateDeleting {
					r.Log.Info("deletion in progress, will requeue")
				} else {
					// Should delete; start async deletion.
					_, err := r.GroupsClient.Delete(ctx, &resourceGroup)
					if err != nil {
						r.Log.Info("error while deleting external resources")
						resultErr = multierror.Append(resultErr, err)
					} else {
						r.Log.Info("started deletion of resource group")
						resourceGroup.Status.ProvisioningState = provisioningStateDeleting
						if err := r.Status().Update(ctx, &resourceGroup); err != nil {
							log.Info("failed to update status after reconcile")
							return ctrl.Result{}, err
						}
					}
				}
				// Requeue while we wait for deletion, returning either/both error(s) if appropriate
				return ctrl.Result{Requeue: true}, resultErr.ErrorOrNil()
			}
			r.Log.Info("finished deletion of resource group")
			// Deletion done; remove our finalizer from the list and update object in API server.
			resourceGroup.ObjectMeta.Finalizers = removeString(resourceGroup.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &resourceGroup); err != nil {
				log.Info("failed to update object after deletion")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Reconcile Azure resources.
	if oldGeneration != resourceGroup.ObjectMeta.GetGeneration() {
		group, err = r.GroupsClient.Ensure(ctx, &resourceGroup)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Attempt to update status in API server.
	if group.Properties != nil && group.Properties.ProvisioningState != nil {
		resourceGroup.Status.ProvisioningState = *group.Properties.ProvisioningState
	}
	if err := r.Status().Update(ctx, &resourceGroup); err != nil {
		log.Info("failed to update status after reconcile")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up this controller for use.
func (r *ResourceGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ResourceGroup{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

//
//
// Helpers below this line
//
//
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
