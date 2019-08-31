/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/keyvaults"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const ManagedAnnotation string = "managed"

// KeyvaultReconciler reconciles a Keyvault object
type KeyvaultReconciler struct {
	client.Client
	config.Config
	Scheme       *runtime.Scheme
	Log          logr.Logger
	VaultsClient *keyvaults.Client
	GroupsClient *resourcegroups.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=keyvaults,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=keyvaults/status,verbs=get;update;patch

func (r *KeyvaultReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("keyvault", req.NamespacedName)

	// Fetch object from Kubernetes API server
	var keyVault azurev1alpha1.Keyvault
	if err := r.Get(ctx, req.NamespacedName, &keyVault); err != nil {
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	var resultErr *multierror.Error

	// Authorize client, don't requeue if fail to instantiate Azure client.
	if err := r.VaultsClient.ForSubscription(keyVault.Spec.SubscriptionID); err != nil {
		resultErr = multierror.Append(resultErr, err)
	}
	if err := r.GroupsClient.ForSubscription(keyVault.Spec.SubscriptionID); err != nil {
		resultErr = multierror.Append(resultErr, err)
	}
	if final := resultErr.ErrorOrNil(); final != nil {
		return ctrl.Result{Requeue: false}, final
	}

	// Check existence of keyvault
	vault, err := r.VaultsClient.Get(ctx, &keyVault)
	if err != nil && !vault.IsHTTPStatus(http.StatusNotFound) {
		return ctrl.Result{}, err
	}
	if vault.IsHTTPStatus(http.StatusNotFound) {
		keyVault.Status.Exists = false
	} else {
		keyVault.Status.Exists = true
	}

	// Update our awareness of the state
	oldGeneration := keyVault.Status.Generation
	keyVault.Status.Generation = keyVault.ObjectMeta.GetGeneration()
	if err := r.Status().Update(ctx, &keyVault); err != nil {
		return ctrl.Result{}, err
	}

	// Handle deletion/finalizer
	if keyVault.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add finalizer if not present
		if !containsString(keyVault.ObjectMeta.Finalizers, finalizerName) {
			keyVault.ObjectMeta.Finalizers = append(keyVault.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &keyVault); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(keyVault.ObjectMeta.Finalizers, finalizerName) {
			if keyVault.Status.Exists {
				// Check if the resource group is managed
				var resourceGroup azurev1alpha1.ResourceGroup
				namespacedName := types.NamespacedName{
					Name:      keyVault.Spec.ResourceGroup,
					Namespace: keyVault.Namespace,
				}
				err := r.Client.Get(ctx, namespacedName, &resourceGroup)
				if client.IgnoreNotFound(err) != nil {
					return ctrl.Result{}, err
				}

				// Delete the managed resource group if necessary
				if err == nil {
					if _, ok := resourceGroup.Annotations[ManagedAnnotation]; ok {
						err = r.Client.Delete(ctx, &resourceGroup, client.PropagationPolicy(metav1.DeletePropagationForeground))
						if err != nil {
							return ctrl.Result{}, err
						}
					}
				}

				var resultErr *multierror.Error
				err = r.VaultsClient.Delete(ctx, &keyVault)
				if err != nil {
					r.Log.Info("error while deleting keyvault in azure")
					resultErr = multierror.Append(resultErr, err)
				} else {
					r.Log.Info("started deletion of keyvault")
				}

				// Try to update status, accumulate errors
				keyVault.Status.Exists = false
				if err := r.Status().Update(ctx, &keyVault); err != nil {
					log.Info("failed to update status after deletion")
					resultErr = multierror.Append(resultErr, err)
				}
				// Requeue while we wait for deletion, returning either/both error(s) if appropriate
				return ctrl.Result{}, resultErr.ErrorOrNil()
			}
			log.Info("finished deletion of keyvault")
			// Deletion done; remove our finalizer from the list and update object in API server.
			keyVault.ObjectMeta.Finalizers = removeString(keyVault.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &keyVault); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Reconcile Azure resources.
	if oldGeneration != keyVault.Status.Generation || !keyVault.Status.Exists {
		resourceGroup := azurev1alpha1.ResourceGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      keyVault.Spec.ResourceGroup,
				Namespace: keyVault.Namespace,
			},
			Spec: azurev1alpha1.ResourceGroupSpec{
				Name: keyVault.Spec.ResourceGroup,
			},
		}

		// *Maybe* resource group exists.
		group, err := r.GroupsClient.Get(ctx, &resourceGroup)
		ok := group.HasHTTPStatus(http.StatusOK, http.StatusNoContent, http.StatusNotFound)
		if err != nil && !ok {
			return ctrl.Result{}, err
		}
		// Create if no group exists
		if group.IsHTTPStatus(http.StatusNotFound) {
			resourceGroup = azurev1alpha1.ResourceGroup{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ManagedAnnotation: "true", // TODO(ace): think of a better strategy for this.
					},
					Name:      keyVault.Spec.ResourceGroup,
					Namespace: keyVault.Namespace,
				},
				Spec: azurev1alpha1.ResourceGroupSpec{
					Name:           keyVault.Spec.ResourceGroup,
					Location:       keyVault.Spec.Location,
					SubscriptionID: keyVault.Spec.SubscriptionID,
				},
			}
			err = controllerutil.SetControllerReference(&keyVault, &resourceGroup, r.Scheme)
			if err != nil {
				return ctrl.Result{}, err
			}
			log.Info("creating resource group")
			if err = r.Create(ctx, &resourceGroup); err != nil {
				log.Info("failed creating resource group")
				return ctrl.Result{}, err
			}
		}
		log.Info("creating keyvault")
		if err := r.VaultsClient.Ensure(ctx, &keyVault); err != nil {
			log.Info("failed creating keyvault")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	log.Info("skipping reconciliation, smooth sailing.")
	return ctrl.Result{}, nil
}

func (r *KeyvaultReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Keyvault{}).
		Owns(&azurev1alpha1.ResourceGroup{}).
		Complete(r)
}
