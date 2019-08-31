/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const (
	finalizerName             string = "azure.alexeldeib.xyz/finalizer"
	provisioningStateDeleting string = "Deleting"
	provisioningStateNotFound string = "NotFound"
)

// ResourceGroupReconciler reconciles a ResourceGroup object
type ResourceGroupReconciler struct {
	client.Client
	config.Config
	Log          logr.Logger
	GroupsClient *resourcegroups.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups/status,verbs=get;update;patch

// Reconcile reconciles a user request for a Resource Group against Azure.
func (r *ResourceGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("resourcegroup", req.NamespacedName)

	// Fetch object from Kubernetes API server
	var resourceGroup azurev1alpha1.ResourceGroup
	if err := r.Get(ctx, req.NamespacedName, &resourceGroup); err != nil {
		log.Info("unable to fetch ResourceGroup")
		return ctrl.Result{}, client.IgnoreNotFound(err)
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

	// The object is being deleted
	if !resourceGroup.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(resourceGroup.ObjectMeta.Finalizers, finalizerName) {
			// We either have 404 or 200. If 200, we may need to begin deletion, wait for deletion to finish,
			if group.IsHTTPStatus(http.StatusOK) {
				// Deletion started; wait for completion.
				if *group.Properties.ProvisioningState == provisioningStateDeleting {
					r.Log.Info("deletion in progress, will requeue")
				} else {
					// Should delete; start async deletion.
					_, err := r.GroupsClient.Delete(ctx, &resourceGroup)
					if err != nil {
						r.Log.Info("error while deleting external resources")
						return ctrl.Result{}, err
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
				return ctrl.Result{Requeue: true}, nil
			}
			r.Log.Info("finished deletion of resource group")
			if err := RemoveFinalizerAndUpdate(ctx, r.Client, finalizerName, &resourceGroup); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := AddFinalizerAndUpdate(ctx, r.Client, finalizerName, &resourceGroup); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile Azure resources.
	if oldGeneration != resourceGroup.ObjectMeta.GetGeneration() || resourceGroup.Status.ProvisioningState == provisioningStateNotFound {
		group, err = r.GroupsClient.Ensure(ctx, &resourceGroup)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		log.Info("skipping reconciliation, smooth sailing.")
	}

	// Attempt to update status in API server, since resource group usually creates immediately.
	// TODO(ace): consider removing this and requeuing request to get status updates on next loop?
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
