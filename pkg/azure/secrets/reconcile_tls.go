/*
Copyright 2019 Alexander Eldeib.
*/

package secrets

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type TLSReconciler struct {
	Service    *SecretService
	Kubeclient ctrl.Client
	*runtime.Scheme
}

// Ensure takes a spec corresponding to one Azure KV secret. It syncs that secret into Kubernetes, remapping the name if necessary.
func (r *TLSReconciler) Ensure(ctx context.Context, obj runtime.Object) error {
	secret, err := r.convert(obj)
	if err != nil {
		return err
	}

	// Fetch base64 encoded PKCS12 secret data w/o password from Azure KV
	bundle, err := r.Service.Get(ctx, secret.Spec.Vault, secret.Spec.Name)
	if err != nil {
		return err
	}

	// Extract the RSA private key from pfx
	tlskey, err := formatRSA(*bundle.Value)
	if err != nil {
		return err
	}

	// Extract the x509 certificate and flip the leaf/root, if necessary
	var tlscert []byte
	if secret.Spec.Reverse {
		tlscert, err = formatX509Reverse(*bundle.Value)
	} else {
		tlscert, err = formatX509Default(*bundle.Value)
	}

	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.ObjectMeta.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Kubeclient, local, func() error {
		if secret.ObjectMeta.UID != "" {
			innerErr := controllerutil.SetControllerReference(secret, local, r.Scheme)
			if innerErr != nil {
				return innerErr
			}
		}
		if local.Data == nil {
			local.Data = map[string][]byte{}
		}
		local.Data["tls.crt"] = tlscert
		local.Data["tls.key"] = tlskey
		local.Type = corev1.SecretTypeTLS
		return nil
	})

	secret.Status.State = nil
	if err == nil {
		secret.Status.State = to.StringPtr("Succeeded")
	}
	return err
}

// Delete deletes a secret from Keyvault.
func (r *TLSReconciler) Delete(ctx context.Context, obj runtime.Object) error {
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

func (r *TLSReconciler) convert(obj runtime.Object) (*azurev1alpha1.TLSSecret, error) {
	local, ok := obj.(*azurev1alpha1.TLSSecret)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
