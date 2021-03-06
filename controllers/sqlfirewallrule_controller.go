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

// SQLFirewallRuleReconciler reconciles a public IP object
type SQLFirewallRuleReconciler struct {
	Reconciler *SyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=sqlfirewallrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=sqlfirewallrules/status,verbs=get;update;patch

func (r *SQLFirewallRuleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.SQLFirewallRule{})
}

func (r *SQLFirewallRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.SQLFirewallRule{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 15}).
		Complete(r)
}
