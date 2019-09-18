/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

// SecurityGroupReconciler reconciles a SecurityGroup object
type SecurityGroupReconciler struct {
	Reconciler *AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=securitygroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=securitygroups/status,verbs=get;update;patch

func (r *SecurityGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.SecurityGroup{})
}

func (r *SecurityGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.SecurityGroup{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
