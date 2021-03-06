/*
Copyright 2019 Alexander Eldeib.
*/

package secrets

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
	kubeclient *ctrl.Client
	scheme     *runtime.Scheme
}

func New(configuration *config.Config, kubeclient *ctrl.Client, scheme *runtime.Scheme) (*Client, error) {
	if kubeclient == nil {
		return nil, errors.New("nil kubeclient passed to secrets client is effectively noop")
	}
	kvclient := keyvault.New()
	authorizer, err := configuration.GetKeyvaultAuthorizer()
	if err != nil {
		return nil, nil
	}
	kvclient.Authorizer = authorizer
	return &Client{internal: kvclient, kubeclient: kubeclient, scheme: scheme}, nil
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	return nil
}

// Get gets a secret from Keyvault.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (keyvault.SecretBundle, error) {
	secret, err := c.convert(obj)
	if err != nil {
		return keyvault.SecretBundle{}, err
	}
	vault := fmt.Sprintf("https://%s.%s", secret.Spec.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
	return c.internal.GetSecret(ctx, vault, secret.Spec.Name, "")
}

// Ensure takes a spec corresponding to one Azure KV secret. It syncs that secret into Kubernetes, remapping the name if necessary.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) error {
	// TODO(ace): cloud-sensitive
	secret, err := c.convert(obj)
	if err != nil {
		return err
	}
	vault := fmt.Sprintf("https://%s.%s", secret.Spec.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
	bundle, err := c.internal.GetSecret(ctx, vault, secret.Spec.Name, "")
	if err != nil {
		return err
	}

	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.ObjectMeta.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, *c.kubeclient, local, func() error {
		if secret.ObjectMeta.UID != "" {
			if ownerErr := controllerutil.SetControllerReference(secret, local, c.scheme); ownerErr != nil {
				return ownerErr
			}
		}
		if local.Data == nil {
			local.Data = map[string][]byte{}
		}
		if secret.Spec.FriendlyName != nil {
			local.Data[*secret.Spec.FriendlyName] = []byte(*bundle.Value)
			return nil
		}
		local.Data[secret.Spec.Name] = []byte(*bundle.Value)
		return nil
	})

	secret.Status.State = nil
	if err == nil {
		secret.Status.State = to.StringPtr("Succeeded")
	}

	return err
}

// Delete deletes a secret from Keyvault.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	local, err := c.convert(obj)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.Name,
			Namespace: local.ObjectMeta.Namespace,
		},
	}
	return client.IgnoreNotFound((*c.kubeclient).Delete(ctx, secret))
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.Secret, error) {
	local, ok := obj.(*azurev1alpha1.Secret)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
