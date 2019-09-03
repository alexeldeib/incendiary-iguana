/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
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
	config.Config
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

	if local.DeletionTimestamp.IsZero() {
		if !HasFinalizer(&local, finalizerName) {
			AddFinalizer(&local, finalizerName)
			if err := r.Update(ctx, &local); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if HasFinalizer(&local, finalizerName) {
			if err := r.GroupsClient.Delete(ctx, &local); err != nil {
				return ctrl.Result{}, err
			}
			RemoveFinalizer(&local, finalizerName)
			if err := r.Update(ctx, &local); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.reconcileExternal(ctx, &local)
}

func (r *ResourceGroupReconciler) reconcileExternal(ctx context.Context, local *azurev1alpha1.ResourceGroup) error {
	// Authorize
	err := r.GroupsClient.ForSubscription(local.Spec.SubscriptionID)
	if err != nil {
		return err
	}

	remote, err := r.GroupsClient.Get(ctx, local)
	if err != nil && !remote.IsHTTPStatus(http.StatusNotFound) {
		return err
	}

	if err := r.updateStatus(ctx, local, remote); err != nil {
		return err
	}

	if err = r.GroupsClient.Ensure(ctx, local); err != nil {
		return err
	}

	if local.Status.ProvisioningState == nil || *local.Status.ProvisioningState != provisioningStateSucceeded {
		return errors.New("not provisioned, should requeue")
	}

	return nil
}

func (r *ResourceGroupReconciler) updateStatus(ctx context.Context, local *azurev1alpha1.ResourceGroup, remote resources.Group) error {
	r.setStatus(ctx, local, remote)
	return r.Status().Update(ctx, local)
}

func (r *ResourceGroupReconciler) setStatus(ctx context.Context, local *azurev1alpha1.ResourceGroup, remote resources.Group) {
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.Properties != nil {
		local.Status.ProvisioningState = remote.Properties.ProvisioningState
	}
}

// SetupWithManager sets up this controller for use.
func (r *ResourceGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ResourceGroup{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
