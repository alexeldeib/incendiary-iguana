/*
Copyright 2019 Alexander Eldeib.
*/

package redis

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers"
)

// RedisReconciler reconciles a Redis object
type RedisController struct {
	Reconciler *reconcilers.AsyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=redis/status,verbs=get;update;patch

// Reconcile reconciles a user request for a Redis cache against Azure.
func (r *RedisController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.Redis{})
}

func (r *RedisController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Redis{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// RedisKeyController reconciles a RedisKey object
type RedisKeyController struct {
	Reconciler *reconcilers.AsyncReconciler
}

// Reconcile reconciles a user request for a Redis cache against Azure.
func (r *RedisKeyController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.Redis{})
}

func (r *RedisKeyController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.RedisKey{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
