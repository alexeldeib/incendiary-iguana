/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// PublicIPReconciler reconciles a public IP object
type PublicIPReconciler struct {
	Reconciler *AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=publicips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=publicips/status,verbs=get;update;patch

func (r *PublicIPReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// ctx := context.Background()
	return r.Reconciler.Reconcile(req, &azurev1alpha1.PublicIP{})
}

func (r *PublicIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.PublicIP{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
