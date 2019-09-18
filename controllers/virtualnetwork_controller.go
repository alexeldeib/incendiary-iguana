/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// VirtualNetworkReconciler reconciles a VirtualNetwork object
type VirtualNetworkReconciler struct {
	Reconciler *AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=virtualnetworks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=virtualnetworks/status,verbs=get;update;patch

// Reconcile reconciles a specification for a virtual network against Azure.
func (r *VirtualNetworkReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.reconciler.Reconcile(req, &azurev1alpha1.VirtualNetwork{})
}

func (r *VirtualNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.VirtualNetwork{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
