/*
Copyright 2019 Alexander Eldeib.
*/

package generic

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/alexeldeib/incendiary-iguana/pkg/constants"
	"github.com/alexeldeib/incendiary-iguana/pkg/finalizer"
)

type SyncActuator interface {
	Ensure(context.Context, runtime.Object) error
	Delete(context.Context, runtime.Object, logr.Logger) error
}

// SyncReconciler is a generic reconciler for synchronous Azure APIs.
type SyncReconciler struct {
	// Kubernetes generic recocniliation components
	client.Client
	logr.Logger
	Recorder record.EventRecorder
	*runtime.Scheme
	// Azure specific reconciliation components
	SyncActuator
}

// NewAsyncReconciler return a new synchronous reconciler for Azure resources associated with the client factory.
func NewSyncReconciler(kubeclient client.Client, act SyncActuator, log logr.Logger, recorder record.EventRecorder, scheme *runtime.Scheme) *SyncReconciler {
	return &SyncReconciler{
		kubeclient,
		log,
		recorder,
		scheme,
		act,
	}
}

func (r *SyncReconciler) Reconcile(req ctrl.Request, local runtime.Object) (ctrl.Result, error) {
	ctx := context.Background()

	gvk, err := apiutil.GVKForObject(local, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	log := r.Logger.WithValues("groupversion", gvk.GroupVersion().String(), "kind", gvk.Kind, "namespace", req.Namespace, "name", req.Name)

	if err := r.Get(ctx, req.NamespacedName, local); err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	res, convertErr := meta.Accessor(local)
	if convertErr != nil {
		return ctrl.Result{}, convertErr
	}

	if res.GetDeletionTimestamp().IsZero() {
		if !finalizer.Has(res, constants.Finalizer) {
			finalizer.Add(res, constants.Finalizer)
			r.Recorder.Event(local, "Normal", "Added", "Object finalizer is added")
			return ctrl.Result{}, r.Update(ctx, local)
		}
	} else {
		if finalizer.Has(res, constants.Finalizer) {
			deleteErr := r.SyncActuator.Delete(ctx, local, log)
			statusErr := r.Status().Update(ctx, local)

			final := multierror.Append(deleteErr, statusErr)
			if err := final.ErrorOrNil(); err != nil {
				r.Recorder.Event(local, "Warning", "FailedDelete", fmt.Sprintf("Failed to delete resource: %s", err.Error()))
				return ctrl.Result{}, err
			}

			r.Recorder.Event(local, "Normal", "Deleted", "Successfully deleted")
			finalizer.Remove(res, constants.Finalizer)
			return ctrl.Result{}, r.Update(ctx, local)
		}
		return ctrl.Result{}, nil
	}

	ensureErr := r.SyncActuator.Ensure(ctx, local)
	statusErr := r.Status().Update(ctx, local)
	if ensureErr != nil {
		log.Error(ensureErr, "ensure err")
	}

	log.Info("successfully reconciled")

	final := multierror.Append(ensureErr, statusErr)
	err = final.ErrorOrNil()
	if err != nil {
		r.Recorder.Event(local, "Warning", "FailedReconcile", fmt.Sprintf("Failed to reconcile resource: %s", err.Error()))
	}

	r.Recorder.Event(local, "Normal", "Reconciled", "Successfully reconciled")
	return ctrl.Result{}, err
}
