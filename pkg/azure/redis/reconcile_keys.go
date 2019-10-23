package redis

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type RedisKeyReconciler struct {
	Service    *RedisService
	Kubeclient ctrl.Client
	Scheme     *runtime.Scheme
}

func (r *RedisKeyReconciler) Ensure(ctx context.Context, obj runtime.Object) error {
	local, err := r.convert(obj)
	if err != nil {
		return err
	}
	if local.Spec.PrimaryKey == nil && local.Spec.SecondaryKey == nil {
		return nil
	}

	keys, err := r.Service.ListKeys(ctx, r.accountFromKey(local))
	if err != nil {
		return err
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.TargetSecret,
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
func (r *RedisKeyReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	local, err := r.convert(obj)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.TargetSecret,
			Namespace: local.ObjectMeta.Namespace,
		},
	}
	return client.IgnoreNotFound(r.Kubeclient.Delete(ctx, secret))
}

func (r *RedisKeyReconciler) convert(obj runtime.Object) (*azurev1alpha1.RedisKey, error) {
	local, ok := obj.(*azurev1alpha1.RedisKey)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}

func (r *RedisKeyReconciler) accountFromKey(key *azurev1alpha1.RedisKey) *azurev1alpha1.Redis {
	return &azurev1alpha1.Redis{
		Spec: azurev1alpha1.RedisSpec{
			SubscriptionID: key.Spec.SubscriptionID,
			ResourceGroup:  key.Spec.ResourceGroup,
			Name:           key.Spec.Name,
		},
	}
}
