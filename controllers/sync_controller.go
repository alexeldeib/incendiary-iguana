/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/sanity-io/litter"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	finalizerName string = "azure.alexeldeib.xyz/finalizer"
)

type SyncClient interface {
	TryAuthorize(context.Context, runtime.Object) error
	TryEnsure(context.Context, runtime.Object) error
	TryDelete(context.Context, runtime.Object) error
}

// SyncReconciler is a generic reconciler for Azure resources which run fast, synchronous operations.
type SyncReconciler struct {
	client.Client
	Az       SyncClient
	Log      logr.Logger
	Recorder record.EventRecorder
}

func (r *SyncReconciler) Reconcile(req ctrl.Request, local runtime.Object) (ctrl.Result, error) {
	ctx := context.Background()
	litter.Dump(local)
	litter.Dump(local.GetObjectKind())
	kind := strings.ToLower(local.GetObjectKind().GroupVersionKind().Kind)
	log := r.Log.WithValues("type", kind, "namespacedName", fmt.Sprintf("%s/%s", req.NamespacedName.Namespace, req.NamespacedName.Name))

	if err := r.Get(ctx, req.NamespacedName, local); err != nil {
		log.Info("error during fetch from api server")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.Az.TryAuthorize(ctx, local); err != nil {
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
			final := multierror.Append(r.Az.TryDelete(ctx, local), r.Status().Update(ctx, local))
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

	final := multierror.Append(r.Az.TryEnsure(ctx, local), r.Status().Update(ctx, local))
	err := final.ErrorOrNil()
	if err != nil {
		r.Recorder.Event(local, "Warning", "FailedReconcile", fmt.Sprintf("Failed to reconcile resource: %s", err.Error()))
	}
	r.Recorder.Event(local, "Normal", "Reconciled", "Successfully reconciled")
	return ctrl.Result{}, err
}
