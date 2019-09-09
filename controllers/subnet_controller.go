/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/subnets"
)

// SubnetReconciler reconciles a Subnet object
type SubnetReconciler struct {
	Reconciler *AzureReconciler
	client.Client
	Log           logr.Logger
	SubnetsClient *subnets.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=subnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=subnets/status,verbs=get;update;patch

func (r *SubnetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var local azurev1alpha1.Subnet

	return r.Reconciler.Reconcile(req, &local)

	// if err := r.Get(ctx, req.NamespacedName, &local); err != nil {
	// 	log.Info("error during fetch from api server")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	// if err := r.SubnetsClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
	// 	return ctrl.Result{}, err
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
	// 		found, err := r.SubnetsClient.Delete(ctx, &local)
	// 		result := multierror.Append(err, r.Status().Update(ctx, &local))
	// 		if err = result.ErrorOrNil(); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 		if !found {
	// 			RemoveFinalizer(&local, finalizerName)
	// 			if err := r.Update(ctx, &local); err != nil {
	// 				return ctrl.Result{}, err
	// 			}
	// 			return ctrl.Result{}, nil
	// 		}
	// 		return ctrl.Result{}, errors.New("requeuing, deletion unfinished")
	// 	}
	// 	return ctrl.Result{}, nil
	// }

	// var final *multierror.Error
	// done, err := r.SubnetsClient.Ensure(ctx, &local)
	// final = multierror.Append(final, err)
	// final = multierror.Append(final, r.Status().Update(ctx, &local))

	// if err := final.ErrorOrNil(); err != nil {
	// 	return ctrl.Result{}, errors.New(final.GoString())
	// }
	// return ctrl.Result{Requeue: !done}, nil
}

func (r *SubnetReconciler) Authorize(obj runtime.Object) error {
	local, ok := obj.(*azurev1alpha1.Subnet)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return r.SubnetsClient.ForSubscription(local.Spec.SubscriptionID)
}
func (r *SubnetReconciler) Ensure(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.Subnet)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return r.SubnetsClient.Ensure(ctx, local)
}
func (r *SubnetReconciler) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	local, ok := obj.(*azurev1alpha1.Subnet)
	if !ok {
		return false, errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return r.SubnetsClient.Delete(ctx, local)
}

func (r *SubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Subnet{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
