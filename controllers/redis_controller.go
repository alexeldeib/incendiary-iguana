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
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/redis"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	Reconciler *AzureReconciler
	client.Client
	Log         logr.Logger
	RedisClient *redis.Client
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=redis/status,verbs=get;update;patch

// Reconcile reconciles a user request for a Resource Group against Azure.
func (r *RedisReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.Redis{})
}

// SetupWithManager sets up this controller for use.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Redis{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
