/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
)

const (
	finalizerName              string = "azure.alexeldeib.xyz/finalizer"
	provisioningStateDeleting  string = "Deleting"
	provisioningStateNotFound  string = "NotFound"
	provisioningStateSucceeded string = "Succeeded"
)

// ResourceGroupReconciler reconciles a ResourceGroup object
type ResourceGroupReconciler struct {
	client.Client
	Log          logr.Logger
	GroupsClient *resourcegroups.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups/status,verbs=get;update;patch

// Reconcile reconciles a user request for a Resource Group against Azure.
func (r *ResourceGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("resourcegroup", req.NamespacedName)

	var local azurev1alpha1.ResourceGroup

	if err := r.Get(ctx, req.NamespacedName, &local); err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Authorize
	if err := r.GroupsClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return ctrl.Result{}, err
	}

	if local.DeletionTimestamp.IsZero() {
		if !HasFinalizer(&local, finalizerName) {
			AddFinalizer(&local, finalizerName)
			if err := r.Update(ctx, &local); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if HasFinalizer(&local, finalizerName) {
			found, err := r.GroupsClient.Delete(ctx, &local)
			result := multierror.Append(err, r.Status().Update(ctx, &local))
			if err = result.ErrorOrNil(); err != nil {
				return ctrl.Result{}, err
			}
			if !found {
				RemoveFinalizer(&local, finalizerName)
				if err := r.Update(ctx, &local); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, errors.New("requeuing, deletion unfinished")
		}
		return ctrl.Result{}, nil
	}

	var final *multierror.Error
	done, err := r.GroupsClient.Ensure(ctx, &local)
	final = multierror.Append(final, err)
	final = multierror.Append(final, r.Status().Update(ctx, &local))

	return ctrl.Result{Requeue: !done}, final.ErrorOrNil()
}

// SetupWithManager sets up this controller for use.
func (r *ResourceGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ResourceGroup{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
