/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/servicebus"
)

// ServiceBusNamespaceReconciler reconciles a ServiceBusNamespace object
type ServiceBusNamespaceReconciler struct {
	Reconciler *AsyncReconciler
	client.Client
	Log                       logr.Logger
	ServiceBusNamespaceClient *servicebus.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=servicebus,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=servicebus/status,verbs=get;update;patch

// Reconcile reconciles a user request for a Service Bus namespace against Azure.
func (r *ServiceBusNamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.ServiceBusNamespace{})
}

// SetupWithManager sets up this controller for use.
func (r *ServiceBusNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ServiceBusNamespace{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
