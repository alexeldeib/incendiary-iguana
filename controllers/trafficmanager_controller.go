/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/trafficmanager/mgmt/2018-04-01/trafficmanager"
	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
)

// TrafficManagerReconciler reconciles a PublicIP object
type TrafficManagerReconciler struct {
	client.Client
	Log                   logr.Logger
	TrafficManagersClient trafficmanagers.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers/status,verbs=get;update;patch

func (r *TrafficManagerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("trafficmanager", req.NamespacedName)

	var local azurev1alpha1.TrafficManager
	var remote trafficmanager.Profile
	var requeue bool

	err := r.Get(ctx, req.NamespacedName, &local)
	if err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{Requeue: !apierrs.IsNotFound(err)}, client.IgnoreNotFound(err)
	}

	remote, err = r.fetchRemote(ctx, local)
	if err != nil && !remote.IsHTTPStatus(http.StatusNotFound) {
		return ctrl.Result{}, err
	}

	if err = r.setStatus(ctx, &local, remote); err != nil {
		return ctrl.Result{}, err
	}

	if local.DeletionTimestamp.IsZero() {
		err := AddFinalizerAndUpdate(ctx, r.Client, finalizerName, &local)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		requeue, err = r.deleteRemote(ctx, &local, remote, log)
		if requeue || err != nil {
			return ctrl.Result{Requeue: requeue}, err
		}
	}

	requeue, err = r.reconcileRemote(ctx, &local, remote, log)
	return ctrl.Result{Requeue: requeue}, err
}

func (r *TrafficManagerReconciler) fetchRemote(ctx context.Context, local azurev1alpha1.TrafficManager) (trafficmanager.Profile, error) {
	// Authorize
	err := r.TrafficManagersClient.ForSubscription(local.Spec.SubscriptionID)
	if err != nil {
		return trafficmanager.Profile{}, err
	}

	return r.TrafficManagersClient.Get(ctx, &local)
}

func (r *TrafficManagerReconciler) setStatus(ctx context.Context, local *azurev1alpha1.TrafficManager, remote trafficmanager.Profile) error {
	if !remote.IsHTTPStatus(http.StatusNotFound) {
		if remote.ID != nil {
			local.Status.ID = *remote.ID
		}
	}
	return r.Status().Update(ctx, local)
}

func (r *TrafficManagerReconciler) reconcileRemote(ctx context.Context, local *azurev1alpha1.TrafficManager, remote trafficmanager.Profile, log logr.Logger) (bool, error) {
	log.Info("reconciling")
	err := r.TrafficManagersClient.Ensure(ctx, local, remote)
	if err != nil {
		return true, err
	}
	return false, nil
}

func (r *TrafficManagerReconciler) deleteRemote(ctx context.Context, local *azurev1alpha1.TrafficManager, remote trafficmanager.Profile, log logr.Logger) (bool, error) {
	if contains(local.ObjectMeta.Finalizers, finalizerName) {
		if remote.IsHTTPStatus(http.StatusNotFound) {
			log.Info("deletion complete")
			return true, RemoveFinalizerAndUpdate(ctx, r.Client, finalizerName, local)
		}
		err := r.TrafficManagersClient.Delete(ctx, local)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	log.Info("no finalizer, not handling deletion")
	return false, nil
}

func (r *TrafficManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.TrafficManager{}).
		Complete(r)
}
