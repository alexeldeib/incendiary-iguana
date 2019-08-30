/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-04-01/network"
	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/nics"
)

// NetworkInterfaceReconciler reconciles a NetworkInterface object
type NetworkInterfaceReconciler struct {
	client.Client
	Log        logr.Logger
	NICsClient nics.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=networkinterfaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=networkinterfaces/status,verbs=get;update;patch

func (r *NetworkInterfaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("networkinterface", req.NamespacedName)

	var local azurev1alpha1.NetworkInterface
	var remote network.Interface
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

	lastReconciled := r.setStatus(ctx, &local, remote)
	err = r.Status().Update(ctx, &local)
	if err != nil {
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

	requeue, err = r.reconcileRemote(ctx, lastReconciled, &local, remote, log)
	return ctrl.Result{Requeue: requeue}, err
}

func (r *NetworkInterfaceReconciler) fetchRemote(ctx context.Context, local azurev1alpha1.NetworkInterface) (network.Interface, error) {
	// Authorize
	err := r.NICsClient.ForSubscription(local.Spec.SubscriptionID)
	if err != nil {
		return network.Interface{}, err
	}

	return r.NICsClient.Get(ctx, &local)
}

func (r *NetworkInterfaceReconciler) setStatus(ctx context.Context, local *azurev1alpha1.NetworkInterface, remote network.Interface) int64 {
	if !remote.IsHTTPStatus(http.StatusNotFound) {
		if remote.ProvisioningState != nil {
			local.Status.ProvisioningState = *remote.ProvisioningState
		}
		if remote.ID != nil {
			local.Status.ID = *remote.ID
		}
	}
	old := local.Status.ObservedGeneration
	local.Status.ObservedGeneration = local.ObjectMeta.Generation
	return old
}

func (r *NetworkInterfaceReconciler) reconcileRemote(ctx context.Context, lastReconciled int64, local *azurev1alpha1.NetworkInterface, remote network.Interface, log logr.Logger) (bool, error) {
	log = log.WithValues("rg", local.Spec.ResourceGroup)
	requeue := r.shouldRequeue(lastReconciled, local, remote)
	if requeue {
		log.Info("not done reconciling, will requeue")
		return true, nil
	}

	log.Info("reconciling")
	err := r.NICsClient.Ensure(ctx, local)
	if err != nil {
		return true, err
	}
	return false, nil
}

func (r *NetworkInterfaceReconciler) shouldRequeue(lastReconciled int64, local *azurev1alpha1.NetworkInterface, remote network.Interface) bool {
	if local.Status.ProvisioningState != "" && local.Status.ProvisioningState != "Succeeded" {
		return true
	}
	return false
}

func (r *NetworkInterfaceReconciler) deleteRemote(ctx context.Context, local *azurev1alpha1.NetworkInterface, remote network.Interface, log logr.Logger) (bool, error) {
	if contains(local.ObjectMeta.Finalizers, finalizerName) {
		if remote.IsHTTPStatus(http.StatusNotFound) {
			log.Info("deletion complete")
			return true, RemoveFinalizerAndUpdate(ctx, r.Client, finalizerName, local)
		}
		if remote.IsHTTPStatus(http.StatusOK) && *remote.ProvisioningState == provisioningStateDeleting {
			log.Info("deletion in progress, will requeue")
			return true, nil
		}
		err := r.NICsClient.Delete(ctx, local)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	log.Info("no finalizer, not handling deletion")
	return false, nil
}

func (r *NetworkInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.NetworkInterface{}).
		Complete(r)
}
