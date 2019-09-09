/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/securitygroups"
)

// SecurityGroupReconciler reconciles a SecurityGroup object
type SecurityGroupReconciler struct {
	Reconciler *AzureReconciler
	client.Client
	Log                  logr.Logger
	SecurityGroupsClient *securitygroups.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=securitygroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=securitygroups/status,verbs=get;update;patch

func (r *SecurityGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// ctx := context.Background()
	// log := r.Log.WithValues("securitygroup", req.NamespacedName)

	var local azurev1alpha1.SecurityGroup

	return r.Reconciler.Reconcile(req, &local)

	// if err := r.SecurityGroupsClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
	// 	return ctrl.Result{}, err
	// }

	// if err := r.Get(ctx, req.NamespacedName, &local); err != nil {
	// 	log.Info("error during fetch from api server")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	// if err := r.SecurityGroupsClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
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
	// 		found, err := r.SecurityGroupsClient.Delete(ctx, &local)
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
	// done, err := r.SecurityGroupsClient.Ensure(ctx, &local)
	// final = multierror.Append(final, err)
	// final = multierror.Append(final, r.Status().Update(ctx, &local))
	// return ctrl.Result{Requeue: !done}, final.ErrorOrNil()
}

func (r *SecurityGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.SecurityGroup{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
