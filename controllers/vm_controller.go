/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// VMReconciler reconciles a VM object
type VMReconciler struct {
	Reconciler *AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=vms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=vms/status,verbs=get;update;patch

// Reconcile reconciles a user request for a virtual machine against Azure.
func (r *VMReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.reconciler.Reconcile(req, &azurev1alpha1.VM{})
}

// SetupWithManager sets up this controller for use.
func (r *VMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.VM{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
