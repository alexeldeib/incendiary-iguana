/*
Copyright 2019 Alexander Eldeib.
*/

package secretbundles

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	internal   keyvault.BaseClient
	kubeclient ctrl.Client
	scheme     *runtime.Scheme
}

func New(configuration *config.Config, kubeclient ctrl.Client, scheme *runtime.Scheme) (*Client, error) {
	kvclient := keyvault.New()
	authorizer, err := configuration.GetKeyvaultAuthorizer()
	if err != nil {
		return nil, nil
	}
	kvclient.Authorizer = authorizer
	return &Client{internal: kvclient, kubeclient: kubeclient, scheme: scheme}, nil
}

// Get gets a secret from Keyvault.
func (c *Client) Get(ctx context.Context, secret *azurev1alpha1.Secret) (keyvault.SecretBundle, error) {
	vault := fmt.Sprintf("https://%s.%s", secret.Spec.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
	return c.internal.GetSecret(ctx, vault, secret.Spec.Name, "")
}

// Ensure takes a spec corresponding to one Azure KV secret. It syncs that secret into Kubernetes, remapping the name if necessary.
func (c *Client) Ensure(ctx context.Context, secret *azurev1alpha1.SecretBundle) error {
	// TODO(ace): cloud-sensitive
	secrets := map[string]*string{}
	for _, item := range secret.Spec.Secrets {
		vault := fmt.Sprintf("https://%s.%s", item.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
		bundle, err := c.internal.GetSecret(ctx, vault, item.Name, "")
		// TODO(ace): more graceful error handling?
		// parallelize and collect?
		if err != nil {
			return err
		}
		if item.NewName != nil {
			secrets[*item.NewName] = bundle.Value
			continue
		}
		secrets[item.Name] = bundle.Value
	}

	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Spec.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c.kubeclient, local, func() error {
		innerErr := controllerutil.SetControllerReference(secret, secret, c.scheme)
		if innerErr != nil {
			return innerErr
		}
		if local.Data == nil {
			local.Data = map[string][]byte{}
		}
		for key, val := range secrets {
			local.Data[key] = []byte(*val)
		}
		return nil
	})

	if local.Data != nil {
		secret.Status.Secrets = map[string]string{}
		for key := range local.Data {
			secret.Status.Secrets[key] = "Succeeded"
		}
		secret.Status.State = to.StringPtr("Succeeded")
	}

	return err
}

// Delete deletes a secret from Keyvault.
func (c *Client) Delete(ctx context.Context, secret *azurev1alpha1.SecretBundle) error {
	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Spec.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
	}
	return client.IgnoreNotFound(c.kubeclient.Delete(ctx, local))
}

func (c *Client) TryAuthorize(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*azurev1alpha1.SecretBundle)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return nil
}

func (c *Client) TryEnsure(ctx context.Context, obj runtime.Object) error {
	local, ok := obj.(*azurev1alpha1.SecretBundle)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Ensure(ctx, local)
}

func (c *Client) TryDelete(ctx context.Context, obj runtime.Object) error {
	local, ok := obj.(*azurev1alpha1.SecretBundle)
	if !ok {
		return errors.New("attempted to parse wrong object type during reconciliation (dev error)")
	}
	return c.Delete(ctx, local)
}
