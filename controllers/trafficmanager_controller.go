/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
)

// TrafficManagerReconciler reconciles a PublicIP object
type TrafficManagerReconciler struct {
	client.Client
	Log                   logr.Logger
	TrafficManagersClient *trafficmanagers.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers/status,verbs=get;update;patch

func (r *TrafficManagerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("trafficmanager", req.NamespacedName)

	var local azurev1alpha1.TrafficManager

	if err := r.TrafficManagersClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, req.NamespacedName, &local); err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
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
			if err := r.TrafficManagersClient.Delete(ctx, &local); err != nil {
				return ctrl.Result{}, err
			}
			RemoveFinalizer(&local, finalizerName)
			if err := r.Update(ctx, &local); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	var final *multierror.Error
	final = multierror.Append(final, r.TrafficManagersClient.Ensure(ctx, &local))
	final = multierror.Append(final, r.Status().Update(ctx, &local))

	return ctrl.Result{}, final.ErrorOrNil()
}

func (r *TrafficManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.TrafficManager{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
