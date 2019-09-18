/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// IdentityReconciler reconciles a managed identity
type IdentityReconciler struct {
	Reconciler *SyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=identities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=identities/status,verbs=get;update;patch

func (r *IdentityReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.Identity{})
}

func (r *IdentityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Identity{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
