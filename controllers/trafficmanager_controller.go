/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	"k8s.io/client-go/tools/record"
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
	Recorder              record.EventRecorder
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers/status,verbs=get;update;patch

func (r *TrafficManagerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("trafficmanager", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, req.NamespacedName.Name))

	var local azurev1alpha1.TrafficManager

	if err := r.Get(ctx, req.NamespacedName, &local); err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.TrafficManagersClient.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return ctrl.Result{}, err
	}

	if local.DeletionTimestamp.IsZero() {
		if !HasFinalizer(&local, finalizerName) {
			AddFinalizer(&local, finalizerName)
			r.Recorder.Event(&local, "Normal", "Added", "Object finalizer is added")
			return ctrl.Result{}, r.Update(ctx, &local)
		}
	} else {
		if HasFinalizer(&local, finalizerName) {
			err := multierror.Append(r.TrafficManagersClient.Delete(ctx, &local), r.Status().Update(ctx, &local))
			if final := err.ErrorOrNil(); final != nil {
				r.Recorder.Event(&local, "Warning", "FailedDelete", fmt.Sprintf("Failed to delete resource: %s", final.Error()))
				return ctrl.Result{}, final
			}
			r.Recorder.Event(&local, "Normal", "Deleted", "Successfully deleted")
			RemoveFinalizer(&local, finalizerName)
			if err := r.Update(ctx, &local); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	done, ensureErr := r.TrafficManagersClient.Ensure(ctx, &local)
	final := multierror.Append(ensureErr, r.Status().Update(ctx, &local))
	err := final.ErrorOrNil()
	if err != nil {
		r.Recorder.Event(&local, "Warning", "FailedReconcile", fmt.Sprintf("Failed to reconcile resource: %s", err.Error()))
	} else if done {
		r.Recorder.Event(&local, "Normal", "Reconciled", "Successfully reconciled")
	}
	return ctrl.Result{Requeue: !done}, err
}

func (r *TrafficManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.TrafficManager{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
