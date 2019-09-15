/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/keyvaults"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const ManagedAnnotation string = "managed"

// KeyvaultReconciler reconciles a Keyvault object
type KeyvaultReconciler struct {
	client.Client
	*config.Config
	Log          logr.Logger
	VaultsClient *keyvaults.Client
	Reconciler   *AzureSyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=keyvaults,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=keyvaults/status,verbs=get;update;patch

func (r *KeyvaultReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// ctx := context.Background()
	// log := r.Log.WithValues("keyvault", req.NamespacedName)

	// var local azurev1alpha1.Keyvault

	// if err := r.Get(ctx, req.NamespacedName, &local); err != nil {
	// 	log.Info("error during fetch from api server")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	// if local.DeletionTimestamp.IsZero() {
	// 	if !HasFinalizer(&local, finalizerName) {
	// 		AddFinalizer(&local, finalizerName)
	// 		if err := r.Update(ctx, &local); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// } else {
	// 	if HasFinalizer(&local, finalizerName) {
	// 		if err := r.VaultsClient.Delete(ctx, &local); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 		RemoveFinalizer(&local, finalizerName)
	// 		if err := r.Update(ctx, &local); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// 	return ctrl.Result{}, nil
	// }

	// var final *multierror.Error
	// final = multierror.Append(final, r.reconcileExternal(ctx, &local))
	// final = multierror.Append(final, r.Status().Update(ctx, &local))

	// return ctrl.Result{}, final.ErrorOrNil()
	return r.Reconciler.Reconcile(req, &azurev1alpha1.Keyvault{})
}

func (r *KeyvaultReconciler) reconcileExternal(ctx context.Context, local *azurev1alpha1.Keyvault) error {
	if err := r.VaultsClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return err
	}

	if err := r.VaultsClient.Ensure(ctx, local); err != nil {
		return err
	}

	return nil
}

func (r *KeyvaultReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Keyvault{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
