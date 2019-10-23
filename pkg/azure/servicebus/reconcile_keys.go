/*
Copyright 2019 Alexander Eldeib.
*/

package servicebus

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
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

type ServiceBusKeyReconciler struct {
	Service    ServiceBusService
	Kubeclient ctrl.Client
	Scheme     *runtime.Scheme
}

// Ensure creates or updates a redis cache in an idempotent manner.
func (r *ServiceBusKeyReconciler) Ensure(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	local, err := r.convert(obj)
	if err != nil {
		return err
	}

	if local.Spec.PrimaryKey == nil && local.Spec.SecondaryKey == nil {
		return nil
	}

	keys, err := r.Service.ListKeys(ctx, local)
	if err != nil {
		return err
	}

	data, err := buildSecret(keys, local)
	if err != nil {
		return err
	}

	target := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *local.Spec.TargetSecret,
			Namespace: local.ObjectMeta.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Kubeclient, target, func() error {
		if target.Data == nil {
			target.Data = map[string][]byte{}
		}

		for key, val := range data {
			target.Data[key] = val
		}

		return nil
	})
	return err
}

// Delete handles deletion of a resource groups.
func (r *ServiceBusKeyReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	local, err := r.convert(obj)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *local.Spec.TargetSecret,
			Namespace: local.ObjectMeta.Namespace,
		},
	}
	return client.IgnoreNotFound(r.Kubeclient.Delete(ctx, secret))
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (r *ServiceBusKeyReconciler) SetStatus(local *azurev1alpha1.ServiceBusNamespace, remote servicebus.SBNamespace) {
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.SBNamespaceProperties != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
}

func (r *ServiceBusKeyReconciler) convert(obj runtime.Object) (*azurev1alpha1.ServiceBusNamespace, error) {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}

func buildSecret(keys servicebus.AccessKeys, local *azurev1alpha1.ServiceBusNamespace) (map[string][]byte, error) {
	var final error
	result := map[string][]byte{}
	if local.Spec.PrimaryKey != nil {
		if keys.PrimaryKey != nil {
			result[*local.Spec.PrimaryKey] = []byte(*keys.PrimaryKey)
		} else {
			final = multierror.Append(final, errors.New("expected primary key but found nil"))
		}
	}

	if local.Spec.SecondaryKey != nil {
		if keys.SecondaryKey != nil {
			result[*local.Spec.SecondaryKey] = []byte(*keys.SecondaryKey)
		} else {
			final = multierror.Append(final, errors.New("expected primary key but found nil"))
		}
	}

	if local.Spec.PrimaryConnectionString != nil {
		if keys.PrimaryConnectionString != nil {
			result[*local.Spec.PrimaryConnectionString] = []byte(*keys.PrimaryConnectionString)
		} else {
			final = multierror.Append(final, errors.New("expected primary key but found nil"))
		}
	}

	if local.Spec.SecondaryConnectionString != nil {
		if keys.SecondaryConnectionString != nil {
			result[*local.Spec.SecondaryConnectionString] = []byte(*keys.SecondaryConnectionString)
		} else {
			final = multierror.Append(final, errors.New("expected primary key but found nil"))
		}
	}
	return result, final
}
