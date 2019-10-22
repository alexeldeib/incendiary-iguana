/*
Copyright 2019 Alexander Eldeib.
*/

package reconcilers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services"
)

type RedisKeyReconciler struct {
	Service    *services.RedisService
	Kubeclient ctrl.Client
	Scheme     *runtime.Scheme
}

func (r *RedisKeyReconciler) SyncSecrets(ctx context.Context, local *azurev1alpha1.Redis) error {
	if local.Spec.PrimaryKey == nil && local.Spec.SecondaryKey == nil {
		return nil
	}

	keys, err := r.Service.ListKeys(ctx, local)
	if err != nil {
		return err
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *local.Spec.TargetSecret,
			Namespace: local.ObjectMeta.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Kubeclient, targetSecret, func() error {
		var final *multierror.Error

		if targetSecret.Data == nil {
			targetSecret.Data = map[string][]byte{}
		}

		if local.ObjectMeta.UID != "" {
			final = multierror.Append(final, controllerutil.SetControllerReference(local, targetSecret, r.Scheme))
		}

		if local.Spec.PrimaryKey != nil {
			if keys.PrimaryKey != nil {
				targetSecret.Data[*local.Spec.PrimaryKey] = []byte(*keys.PrimaryKey)
			} else {
				final = multierror.Append(final, errors.New("expected primary key but found nil"))
			}
		}

		if local.Spec.SecondaryKey != nil {
			if keys.SecondaryKey != nil {
				targetSecret.Data[*local.Spec.SecondaryKey] = []byte(*keys.SecondaryKey)
			} else {
				final = multierror.Append(final, errors.New("expected secondary key but found nil"))
			}
		}

		return final.ErrorOrNil()
	})
	return err
}

// Delete handles deletion of a resource groups.
func (r *RedisKeyReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := r.convert(obj)
	if err != nil {
		return false, err
	}
	return r.Service.Delete(ctx, local, log)
}