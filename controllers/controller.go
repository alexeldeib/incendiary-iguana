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

// AzureReconciler is a generic reconciler for Azure objects
type AzureReconciler struct {
	client.Client
	Az       AzureClient
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

	res, err := meta.Accessor(local)
	if err != nil {
		return ctrl.Result{}, err
	}

	if res.GetDeletionTimestamp().IsZero() {
		if !HasFinalizer(res, finalizerName) {
			AddFinalizer(res, finalizerName)
			r.Recorder.Event(local, "Normal", "Added", "Object finalizer is added")
			return ctrl.Result{}, r.Update(ctx, local)
		}
	} else {
		if HasFinalizer(res, finalizerName) {
			found, err := r.Az.TryDelete(ctx, local)
			result := multierror.Append(err, r.Status().Update(ctx, local))
			if err = result.ErrorOrNil(); err != nil {
				r.Recorder.Event(local, "Warning", "FailedDelete", "Failed to delete resource")
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

	var final *multierror.Error
	done, err := r.Az.TryEnsure(ctx, local)
	final = multierror.Append(final, err)
	final = multierror.Append(final, r.Status().Update(ctx, local))
	err = final.ErrorOrNil()
	if err != nil {
		r.Recorder.Event(local, "Warning", "FailedReconcile", "Failed to reconcile resource")
	} else if done {
		r.Recorder.Event(local, "Normal", "Reconciled", "Successfully reconciled")
	}
	return ctrl.Result{Requeue: !done}, err
}
