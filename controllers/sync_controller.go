/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

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
)

const (
	finalizerName string = "azure.alexeldeib.xyz/finalizer"
)

type SyncClient interface {
	ForSubscription(context.Context, runtime.Object) error
	Ensure(context.Context, runtime.Object) error
	Delete(context.Context, runtime.Object) error
}

// SyncReconciler is a generic reconciler for Azure resources which run fast, synchronous operations.
type SyncReconciler struct {
	client.Client
	Az       SyncClient
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *SyncReconciler) Reconcile(req ctrl.Request, local runtime.Object) (ctrl.Result, error) {
	ctx := context.Background()

	gvk, err := apiutil.GVKForObject(local, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	log := r.Log.WithValues("groupversion", gvk.GroupVersion().String(), "kind", gvk.Kind, "namespace", req.Namespace, "name", req.Name)

	if err := r.Get(ctx, req.NamespacedName, local); err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.Az.ForSubscription(ctx, local); err != nil {
		return ctrl.Result{}, err
	}

	res, convertErr := meta.Accessor(local)
	if convertErr != nil {
		return ctrl.Result{}, convertErr
	}

	if res.GetDeletionTimestamp().IsZero() {
		if !HasFinalizer(res, finalizerName) {
			AddFinalizer(res, finalizerName)
			r.Recorder.Event(local, "Normal", "Added", "Object finalizer is added")
			return ctrl.Result{}, r.Update(ctx, local)
		}
	} else {
		if HasFinalizer(res, finalizerName) {
			final := multierror.Append(r.Az.Delete(ctx, local), r.Status().Update(ctx, local))
			if err := final.ErrorOrNil(); err != nil {
				r.Recorder.Event(local, "Warning", "FailedDelete", fmt.Sprintf("Failed to delete resource: %s", err.Error()))
				return ctrl.Result{}, err
			}
			r.Recorder.Event(local, "Normal", "Deleted", "Successfully deleted")
			RemoveFinalizer(res, finalizerName)
			return ctrl.Result{}, r.Update(ctx, local)
		}
		return ctrl.Result{}, nil
	}
	ensureErr := r.Az.Ensure(ctx, local)
	if ensureErr != nil {
		log.Error(ensureErr, "ensure err")
	}
	log.Info("successfully reconciled")
	final := multierror.Append(ensureErr, r.Status().Update(ctx, local))
	err = final.ErrorOrNil()
	if err != nil {
		r.Recorder.Event(local, "Warning", "FailedReconcile", fmt.Sprintf("Failed to reconcile resource: %s", err.Error()))
	}
	r.Recorder.Event(local, "Normal", "Reconciled", "Successfully reconciled")
	return ctrl.Result{}, err
}
