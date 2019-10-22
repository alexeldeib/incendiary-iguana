/*
Copyright 2019 Alexander Eldeib.
*/

package reconcilers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services"
)

type SecretReconciler struct {
	Service    *services.SecretService
	Kubeclient ctrl.Client
	Scheme     *runtime.Scheme
}

// Ensure takes a spec corresponding to several Azure KV secrets. It syncs them into Kubernetes, remapping the names as necessary.
func (r *SecretReconciler) Ensure(ctx context.Context, obj runtime.Object) error {
	secret, err := r.convert(obj)
	if err != nil {
		return err
	}

	out, err := downloadAndFormat(ctx, r.Service, secret.Spec.SecretIdentifier)
	if err != nil {
		return errors.Wrapf(err, "failed to download and format secret %s", secret.Spec.SecretIdentifier.Name)
	}

	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.ObjectMeta.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Kubeclient, local, func() error {
		if secret.ObjectMeta.UID != "" {
			if ownerErr := controllerutil.SetControllerReference(secret, local, r.Scheme); ownerErr != nil {
				return ownerErr
			}
		}
		if local.Data == nil {
			local.Data = map[string][]byte{}
		}
		local.Data[secret.Spec.TargetValue] = out
		return nil
	})

	return err
}

// Delete deletes a secret from Keyvault.
func (r *SecretReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	secret, err := r.convert(obj)
	if err != nil {
		return err
	}
	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Spec.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
	}
	return client.IgnoreNotFound(r.Kubeclient.Delete(ctx, local))
}

func (r *SecretReconciler) convert(obj runtime.Object) (*azurev1alpha1.Secret, error) {
	local, ok := obj.(*azurev1alpha1.Secret)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
