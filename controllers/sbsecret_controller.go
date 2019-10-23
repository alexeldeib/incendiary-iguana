/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers/generic"
)

// ServiceBusSecretReconciler reconciles a ServiceBusNamespace object
type ServiceBusSecretController struct {
	Reconciler *generic.AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=servicebus,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=servicebus/status,verbs=get;update;patch

// Reconcile reconciles a user request for a Service Bus namespace against Azure.
func (r *ServiceBusSecretController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.ServiceBusNamespace{})
}

// SetupWithManager sets up this controller for use.
func (r *ServiceBusSecretController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ServiceBusKey{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
