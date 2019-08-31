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
	var remote network.VirtualNetwork
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

func (r *VirtualNetworkReconciler) fetchRemote(ctx context.Context, local azurev1alpha1.VirtualNetwork) (network.VirtualNetwork, error) {
	// Authorize
	err := r.VnetsClient.ForSubscription(local.Spec.SubscriptionID)
	if err != nil {
		return network.VirtualNetwork{}, err
	}

	return r.VnetsClient.Get(ctx, &local)
}

func (r *VirtualNetworkReconciler) setStatus(ctx context.Context, local *azurev1alpha1.VirtualNetwork, remote network.VirtualNetwork) int64 {
	if remote.IsHTTPStatus(http.StatusOK) {
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

func (r *VirtualNetworkReconciler) reconcileRemote(ctx context.Context, lastReconciled int64, local *azurev1alpha1.VirtualNetwork, remote network.VirtualNetwork, log logr.Logger) (bool, error) {
	log = log.WithValues("rg", local.Spec.ResourceGroup, "vnet", local.Spec.Name)
	if remote.IsHTTPStatus(http.StatusOK) && *remote.ProvisioningState != "Succeeded" {
		return true, nil
	}
	if remote.IsHTTPStatus(http.StatusNotFound) || lastReconciled != local.ObjectMeta.Generation {
		log.Info("reconciling virtual network")
		err := r.VnetsClient.Ensure(ctx, local)
		return true, err
	}
	log.Info("skipping reconciliation, smooth sailing")
	return false, nil
}

func (r *VirtualNetworkReconciler) deleteRemote(ctx context.Context, local *azurev1alpha1.VirtualNetwork, remote network.VirtualNetwork, log logr.Logger) (bool, error) {
	if contains(local.ObjectMeta.Finalizers, finalizerName) {
		if remote.IsHTTPStatus(http.StatusNotFound) {
			log.Info("deletion complete")
			return true, RemoveFinalizerAndUpdate(ctx, r.Client, finalizerName, local)
		}
		if remote.IsHTTPStatus(http.StatusOK) && *remote.ProvisioningState == provisioningStateDeleting {
			log.Info("deletion in progress, will requeue")
			return true, nil
		}
		err := r.VnetsClient.Delete(ctx, local)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	log.Info("no finalizer, not handling deletion")
	return false, nil
}

func (r *VirtualNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.VirtualNetwork{}).
		Complete(r)
}
