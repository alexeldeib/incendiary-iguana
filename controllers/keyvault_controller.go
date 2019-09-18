/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

const ManagedAnnotation string = "managed"

// KeyvaultReconciler reconciles a Keyvault object
type KeyvaultReconciler struct {
	Reconciler *SyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=keyvaults,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=keyvaults/status,verbs=get;update;patch

func (r *KeyvaultReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.Keyvault{})
}

func (r *KeyvaultReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Keyvault{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
