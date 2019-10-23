/*
Copyright 2019 Alexander Eldeib.
*/

package resourcegroups

import (
	ctrl "sigs.k8s.io/controller-runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers"
)

// ResourceGroupController reconciles a ResourceGroup object
type ResourceGroupController struct {
	Reconciler *reconcilers.AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=resourcegroups/status,verbs=get;update;patch

// Reconcile reconciles a user request for a Resource Group against Azure.
func (r *ResourceGroupController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.ResourceGroup{})
}

// SetupWithManager sets up this controller for use.
func (r *ResourceGroupController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ResourceGroup{}).
		Complete(r)
}
