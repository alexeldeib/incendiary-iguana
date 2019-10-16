/*
Copyright 2019 Alexander Eldeib.
*/

package dockercfg

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
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
		return nil, err
	}
	kvclient.Authorizer = authorizer
	return &Client{internal: kvclient, kubeclient: kubeclient, scheme: scheme}, nil
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	return nil
}

// Get gets a secret from Keyvault.
func (c *Client) Get(ctx context.Context, obj runtime.Object) (*[]byte, error) {
	secret, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	vault := fmt.Sprintf("https://%s.%s", secret.Spec.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
	passwordBundle, err := c.internal.GetSecret(ctx, vault, secret.Spec.Password, "")
	if err != nil {
		return nil, err
	}

	dockercfgJSONContent, err := handleDockerCfgJSONContent(secret.Spec.Username, *passwordBundle.Value, secret.Spec.Email, secret.Spec.Server)
	if err != nil {
		return nil, err
	}
	return &dockercfgJSONContent, nil
}

// Ensure takes a spec corresponding to one Azure KV secret. It syncs that secret into Kubernetes, remapping the name if necessary.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) error {
	// TODO(ace): cloud-sensitive
	secret, err := c.convert(obj)
	if err != nil {
		return err
	}
	vault := fmt.Sprintf("https://%s.%s", secret.Spec.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
	passwordBundle, err := c.internal.GetSecret(ctx, vault, secret.Spec.Password, "")
	if err != nil {
		return err
	}

	dockercfgJSONContent, err := handleDockerCfgJSONContent(secret.Spec.Username, *passwordBundle.Value, secret.Spec.Email, secret.Spec.Server)
	if err != nil {
		return err
	}

	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.ObjectMeta.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	_, err = controllerutil.CreateOrUpdate(ctx, *c.kubeclient, local, func() error {
		if secret.ObjectMeta.UID != "" {
			if ownerErr := controllerutil.SetControllerReference(secret, local, c.scheme); ownerErr != nil {
				return ownerErr
			}
		}
		local.Data = map[string][]byte{
			corev1.DockerConfigJsonKey: dockercfgJSONContent,
		}
		return nil
	})

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
			Name:      local.ObjectMeta.Name,
			Namespace: local.ObjectMeta.Namespace,
		},
	}
	return client.IgnoreNotFound((*c.kubeclient).Delete(ctx, secret))
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.DockerConfig, error) {
	local, ok := obj.(*azurev1alpha1.DockerConfig)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}

/*
**
** lifted liberally from https://github.com/kubernetes/kubectl/blob/2b775f287352b3191b0e52cc4231941cd8e337bf/pkg/generate/versioned/secret_for_docker_registry.go#L150-L164
**
 */

// handleDockerCfgJSONContent serializes a ~/.docker/config.json file
func handleDockerCfgJSONContent(username, password, email, server string) ([]byte, error) {
	dockercfgAuth := DockerConfigEntry{
		Username: username,
		Password: password,
		Email:    email,
		Auth:     encodeDockerConfigFieldAuth(username, password),
	}

	dockerCfgJSON := DockerConfigJSON{
		Auths: map[string]DockerConfigEntry{server: dockercfgAuth},
	}

	return json.Marshal(dockerCfgJSON)
}

func encodeDockerConfigFieldAuth(username, password string) string {
	fieldValue := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(fieldValue))
}

// DockerConfigJSON represents a local docker auth config file
// for pulling images.
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
	// +optional
	HttpHeaders map[string]string `json:"HttpHeaders,omitempty"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

type DockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

/*
 *
 * end taken from kubetcl
 *
 */
