/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Fetch retrieves an object by namespaced name from the API server and puts the contents in the runtime.Object parameter.
// TODO(ace): refactor onto base reconciler struct
func Fetch(ctx context.Context, client client.Client, namespacedName types.NamespacedName, obj runtime.Object, log logr.Logger) error {
	if err := client.Get(ctx, namespacedName, obj); err != nil {
		// dont't requeue not found
		if apierrs.IsNotFound(err) {
			return nil
		}
		log.Error(err, "unable to fetch secret")
		return err
	}
	return nil
}

func DeleteIfFound(ctx context.Context, client client.Client, obj runtime.Object) error {
	if err := client.Delete(ctx, obj); err != nil && !apierrs.IsNotFound(err) {
		return err
	}
	return nil
}
