/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// StorageKeyReconciler reconciles a storage key object
type StorageKeyReconciler struct {
	Reconciler *SyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=storagekeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=storagekeys/status,verbs=get;update;patch

func (r *StorageKeyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.StorageKey{})
}

func (r *StorageKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.StorageKey{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
