/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	finalizerName              string = "azure.alexeldeib.xyz/finalizer"
	provisioningStateDeleting  string = "Deleting"
	provisioningStateNotFound  string = "NotFound"
	provisioningStateSucceeded string = "Succeeded"
)

type AzureClient interface {
	TryAuthorize(context.Context, runtime.Object) error
	TryEnsure(context.Context, runtime.Object) (bool, error)
	TryDelete(context.Context, runtime.Object) (bool, error)
}

type AzureSyncClient interface {
	TryAuthorize(context.Context, runtime.Object) error
	TryEnsure(context.Context, runtime.Object) error
	TryDelete(context.Context, runtime.Object) error
}

// AzureReconciler is a generic reconciler for Azure objects
type AzureReconciler struct {
	client.Client
	Az       AzureClient
	Log      logr.Logger
	Recorder record.EventRecorder
}

// AzureSyncReconciler is a generic reconciler for Azure resources which run fast, synchronous operations.
type AzureSyncReconciler struct {
	client.Client
	Az       AzureSyncClient
	Log      logr.Logger
	Recorder record.EventRecorder
}

func (r *AzureReconciler) Reconcile(req ctrl.Request, local runtime.Object) (ctrl.Result, error) {
	ctx := context.Background()
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
			found, deleteErr := r.Az.TryDelete(ctx, local)
			final := multierror.Append(deleteErr, r.Status().Update(ctx, local))
			if err := final.ErrorOrNil(); err != nil {
				r.Recorder.Event(local, "Warning", "FailedDelete", fmt.Sprintf("Failed to delete resource: %s", err.Error()))
				return ctrl.Result{}, err
			}
			if !found {
				r.Recorder.Event(local, "Normal", "Deleted", "Successfully deleted")
				RemoveFinalizer(res, finalizerName)
				return ctrl.Result{}, r.Update(ctx, local)
			}
			return ctrl.Result{}, errors.New("requeuing, deletion unfinished")
		}
		return ctrl.Result{}, nil
	}

	done, ensureErr := r.Az.TryEnsure(ctx, local)
	final := multierror.Append(ensureErr, r.Status().Update(ctx, local))
	err := final.ErrorOrNil()
	if err != nil {
		r.Recorder.Event(local, "Warning", "FailedReconcile", fmt.Sprintf("Failed to reconcile resource: %s", err.Error()))
	} else if done {
		r.Recorder.Event(local, "Normal", "Reconciled", "Successfully reconciled")
	}
	return ctrl.Result{Requeue: !done}, err
}

func (r *AzureSyncReconciler) Reconcile(req ctrl.Request, local runtime.Object) (ctrl.Result, error) {
	ctx := context.Background()
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
