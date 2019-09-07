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
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/virtualnetworks"
)

// VirtualNetworkReconciler reconciles a VirtualNetwork object
type VirtualNetworkReconciler struct {
	client.Client
	Log         logr.Logger
	VnetsClient *virtualnetworks.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=virtualnetworks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=virtualnetworks/status,verbs=get;update;patch

func (r *VirtualNetworkReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("virtualnetwork", req.NamespacedName)

	var local azurev1alpha1.VirtualNetwork

	if err := r.Get(ctx, req.NamespacedName, &local); err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Authorize
	if err := r.VnetsClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
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
			found, err := r.VnetsClient.Delete(ctx, &local)
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
	final = multierror.Append(final, r.VnetsClient.Ensure(ctx, &local))
	final = multierror.Append(final, r.Status().Update(ctx, &local))

	return ctrl.Result{}, final.ErrorOrNil()
}

func (r *VirtualNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.VirtualNetwork{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
