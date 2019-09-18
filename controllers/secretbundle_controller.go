/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"regexp"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

var BadHostRegex = regexp.MustCompile(`StatusCode=([0-9]{0,3})`)

// SecretBundleReconciler reconciles a Secret object
type SecretBundleReconciler struct {
	Reconciler *SyncReconciler
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=secretbundles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=secretbundles/status,verbs=get;update;patch

func (r *SecretBundleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &azurev1alpha1.SecretBundle{})
}

func (r *SecretBundleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.SecretBundle{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
