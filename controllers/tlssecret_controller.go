/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// TLSSecretReconciler reconciles a Secret object
type TLSSecretReconciler struct {
	Reconciler *SyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=tlssecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=tlssecrets/status,verbs=get;update;patch

func (r *TLSSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.Secret{})
}

func (r *TLSSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.TLSSecret{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
