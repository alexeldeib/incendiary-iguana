/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// NetworkInterfaceReconciler reconciles a nic object
type NetworkInterfaceReconciler struct {
	Reconciler *AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=networkinterfaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=networkinterfaces/status,verbs=get;update;patch

// Reconcile reconciles a user request for a nic against Azure.
func (r *NetworkInterfaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.NetworkInterface{})
}

// SetupWithManager sets up this controller for use.
func (r *NetworkInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.NetworkInterface{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
