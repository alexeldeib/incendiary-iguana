/*
Copyright 2019 Alexander Eldeib.
*/

package sql

// import (
// 	ctrl "sigs.k8s.io/controller-runtime"

// 	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
// 	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers"
// )

// // SQLServerController reconciles a SQLServer object
// type SQLServerController struct {
// 	Reconciler *reconcilers.AsyncReconciler
// }

// // +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=sqlservers,verbs=get;list;watch;create;update;patch;delete
// // +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=sqlservers/status,verbs=get;update;patch
// func (r *SQLServerController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
// 	return r.Reconciler.Reconcile(req, &azurev1alpha1.SQLServer{})
// }

// func (r *SQLServerController) SetupWithManager(mgr ctrl.Manager) error {
// 	return ctrl.NewControllerManagedBy(mgr).
// 		For(&azurev1alpha1.SQLServer{}).
// 		Complete(r)
// }
