/*
Copyright 2019 Alexander Eldeib.
*/

package generic

import (
	"context"
	"errors"
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

type AsyncActuator interface {
	Ensure(context.Context, runtime.Object) (bool, error)
	Delete(context.Context, runtime.Object, logr.Logger) (bool, error)
}

// AsyncReconciler is a generic reconciler for Azure resources.
// It reconciles object which require long running operations.
type AsyncReconciler struct {
	// Kubernetes generic recocniliation components
	client.Client
	logr.Logger
	Recorder record.EventRecorder
	*runtime.Scheme
	// Azure specific reconciliation components
	// TODO(ace): better naming...
	AsyncActuator
}

// NewAsyncReconciler return a new asynchronous reconciler for Azure resources associated with the client.
func NewAsyncReconciler(kubeclient client.Client, act AsyncActuator, log logr.Logger, recorder record.EventRecorder, scheme *runtime.Scheme) *AsyncReconciler {
	return &AsyncReconciler{
		kubeclient,
		log,
		recorder,
		scheme,
		act,
	}
}

func (r *AsyncReconciler) Reconcile(req ctrl.Request, local runtime.Object) (ctrl.Result, error) {
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

	res, err := meta.Accessor(local)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.V(2).Info("checking deletion timestamp")
	if res.GetDeletionTimestamp().IsZero() {
		log.Info("will try to add finalizer")
		if !finalizer.Has(res, constants.Finalizer) {
			finalizer.Add(res, constants.Finalizer)
			log.Info("added finalizer")
			r.Recorder.Event(local, "Normal", "Added", "Object finalizer is added")
			return ctrl.Result{}, r.Update(ctx, local)
		}
	} else {
		log.V(2).Info("checking for finalizer")
		if finalizer.Has(res, constants.Finalizer) {
			log.V(2).Info("finalizer present, invoking actuator deletion")
			done, deleteErr := r.AsyncActuator.Delete(ctx, local, log)
			statusErr := r.Status().Update(ctx, local)

			final := multierror.Append(deleteErr, statusErr)
			if err := final.ErrorOrNil(); err != nil {
				r.Recorder.Event(local, "Warning", "FailedDelete", fmt.Sprintf("Failed to delete resource: %s", err.Error()))
				return ctrl.Result{}, err
			}

			if done {
				r.Recorder.Event(local, "Normal", "Deleted", "Successfully deleted")
				finalizer.Remove(res, constants.Finalizer)
				return ctrl.Result{}, r.Update(ctx, local)
			}

			return ctrl.Result{}, errors.New("requeuing, deletion unfinished")
		}
		return ctrl.Result{}, nil
	}

	log.V(2).Info("reconciling object")
	done, ensureErr := r.AsyncActuator.Ensure(ctx, local)
	statusErr := r.Status().Update(ctx, local)

	final := multierror.Append(ensureErr, statusErr)
	if err := final.ErrorOrNil(); err != nil {
		log.Error(err, "failed to reconcile")
		r.Recorder.Event(local, "Warning", "FailedReconcile", fmt.Sprintf("Failed to reconcile resource: %s", err.Error()))
	} else if done {
		log.Info("successfully reconciled")
		r.Recorder.Event(local, "Normal", "Reconciled", "Successfully reconciled")
	} else {
		log.Info("reconciled, but will requeue for completion.")
		r.Recorder.Event(local, "Normal", "Reconciled", "Reconciled, but will requeue for completion")
	}

	return ctrl.Result{Requeue: !done}, err
}
