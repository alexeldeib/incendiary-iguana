/*
Copyright 2019 Alexander Eldeib.
*/

package servicebus

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	"github.com/go-logr/logr"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type ServiceBusReconciler struct {
	Service    ServiceBusService
	Kubeclient ctrl.Client
	Scheme     *runtime.Scheme
}

// Ensure creates or updates a redis cache in an idempotent manner.
func (r *ServiceBusReconciler) Ensure(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := r.convert(obj)
	if err != nil {
		return false, err
	}

	remote, err := r.Service.Get(ctx, local)
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	r.SetStatus(local, remote)
	if err != nil && found {
		return false, err
	}

	if found {
		if err := r.SyncSecrets(ctx, local); err != nil {
			return false, err
		}
	}

	return r.Service.CreateOrUpdate(ctx, local, nil)
}

func (r *ServiceBusReconciler) SyncSecrets(ctx context.Context, local *azurev1alpha1.ServiceBusNamespace) error {
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
func (r *ServiceBusReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) (bool, error) {
	local, err := r.convert(obj)
	if err != nil {
		return false, err
	}
	return r.Service.Delete(ctx, local, log)
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (r *ServiceBusReconciler) SetStatus(local *azurev1alpha1.ServiceBusNamespace, remote servicebus.SBNamespace) {
	local.Status.ID = remote.ID
	local.Status.ProvisioningState = nil
	if remote.SBNamespaceProperties != nil {
		local.Status.ProvisioningState = remote.ProvisioningState
	}
}

func (r *ServiceBusReconciler) convert(obj runtime.Object) (*azurev1alpha1.ServiceBusNamespace, error) {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
