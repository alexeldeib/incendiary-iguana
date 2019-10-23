/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers/generic"
)

// TrafficManagerReconciler reconciles a TrafficManager object
type TrafficManagerController struct {
	Reconciler *generic.AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=trafficmanagers/status,verbs=get;update;patch
func (r *TrafficManagerController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.TrafficManager{})
}

func (r *TrafficManagerController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.TrafficManager{}).
		Complete(r)
}
