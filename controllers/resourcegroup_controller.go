/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"
	"time"

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
	Log   logr.Logger
	Azure resourcegroups.Client
}

// AzureResourceGroupReconciler defines methods which call external services for stub generation.
type AzureResourceGroupReconciler interface {
	ReconcileExternalResources(context.Context, *azurev1alpha1.ResourceGroup, resourcegroups.Client, logr.Logger) (ctrl.Result, error)
	DeleteExternalResources(context.Context, *azurev1alpha1.ResourceGroup, resourcegroups.Client) error
}

// NewResourceGroupReconciler instantiates a resource group reconciler with a
// context, Azure credentials, Kubernetes client, and the default Azure client.
func NewResourceGroupReconciler(ctx config.Context) *ResourceGroupReconciler {
	return NewResourceGroupReconcilerForClient(ctx, resourcegroups.New(ctx.Config))
}

// NewResourceGroupReconcilerForClient instantiates a resource group
// reconciler with a context, Azure credentials, Kubernetes client, and
// Azure client.
func NewResourceGroupReconcilerForClient(ctx config.Context, azure resourcegroups.Client) *ResourceGroupReconciler {
	return &ResourceGroupReconciler{
		Config: ctx.Config,
		Client: ctx.Client,
		Log:    ctx.Log.WithName("ResourceGroup"),
		Azure:  azure,
	}
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
	err := r.Azure.ForSubscription(resourceGroup.Spec.SubscriptionID)
	if err != nil {
		// Don't requeue if fail to instantiate Azure client.
		return ctrl.Result{Requeue: false}, err
	}

	// Fetch state of world
	group, err := r.Azure.Get(ctx, &resourceGroup)
	if err != nil && !group.IsHTTPStatus(http.StatusNotFound) {
		return ctrl.Result{}, err
	}

	// Update our awareness of the state
	resourceGroup.Status.Generation = resourceGroup.ObjectMeta.GetGeneration()
	if group.Response.IsHTTPStatus(http.StatusNotFound) {
		resourceGroup.Status.ProvisioningState = provisioningStateNotFound
	}
	if group.Properties != nil {
		resourceGroup.Status.ProvisioningState = *group.Properties.ProvisioningState
	}
	if err := r.Status().Update(ctx, &resourceGroup); err != nil {
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
					if err := r.DeleteExternalResources(ctx, &resourceGroup); err != nil {
						r.Log.Info("error while deleting external resources")
						resultErr = multierror.Append(resultErr, err)
					}
					r.Log.Info("started deletion of resource group")
				}
				// Try to update status, accumulate all errors
				if err := r.Status().Update(ctx, &resourceGroup); err != nil {
					resultErr = multierror.Append(resultErr, err)
				}
				// Requeue while we wait for deletion, returning either/both error(s) if appropriate
				return ctrl.Result{RequeueAfter: time.Second * 5}, resultErr.ErrorOrNil()
			}
			// Deletion done; remove our finalizer from the list and update it.
			resourceGroup.ObjectMeta.Finalizers = removeString(resourceGroup.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &resourceGroup); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Reconcile Azure resources.
	result, err := r.ReconcileExternalResources(ctx, &resourceGroup, log)
	if err != nil {
		return result, err
	}

	// Attempt to update status
	if err := r.Status().Update(ctx, &resourceGroup); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}
	return ctrl.Result{}, nil
}

// ReconcileExternalResources handles reconciliation of Azure resource in a given cloud environment
func (r *ResourceGroupReconciler) ReconcileExternalResources(ctx context.Context, resourceGroup *azurev1alpha1.ResourceGroup, log logr.Logger) (ctrl.Result, error) {
	// Check for existence of Resource group. We only care about location and name.
	// TODO(ace): handle location/name changes? via status somehow
	group, err := r.Azure.Ensure(ctx, resourceGroup)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}
	if group.Properties != nil {
		resourceGroup.Status.ProvisioningState = *group.Properties.ProvisioningState
	}
	return ctrl.Result{Requeue: false}, nil
}

// DeleteExternalResources handles deletion of Azure resource in a given cloud environment
func (r *ResourceGroupReconciler) DeleteExternalResources(ctx context.Context, resourceGroup *azurev1alpha1.ResourceGroup) error {
	status, err := r.Azure.Delete(ctx, resourceGroup)
	if err != nil {
		return err
	}
	resourceGroup.Status.ProvisioningState = status
	return nil
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
