/*
Copyright 2019 Alexander Eldeib.
*/

package secrets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
)

type DockerConfigReconciler struct {
	Service    *SecretService
	Scheme     *runtime.Scheme
	Kubeclient client.Client
}

// Ensure takes a spec corresponding to one Azure KV secret. It syncs that secret into Kubernetes, remapping the name if necessary.
func (c *DockerConfigReconciler) Ensure(ctx context.Context, obj runtime.Object) error {
	// TODO(ace): cloud-sensitive
	secret, err := c.convert(obj)
	if err != nil {
		return err
	}

	passwordBundle, err := c.Service.Get(ctx, secret.Spec.Vault, secret.Spec.Password)
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

	_, err = controllerutil.CreateOrUpdate(ctx, c.Kubeclient, local, func() error {
		if secret.ObjectMeta.UID != "" {
			if ownerErr := controllerutil.SetControllerReference(secret, local, c.Scheme); ownerErr != nil {
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
func (c *DockerConfigReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
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
	return client.IgnoreNotFound(c.Kubeclient.Delete(ctx, secret))
}

func (c *DockerConfigReconciler) convert(obj runtime.Object) (*azurev1alpha1.DockerConfig, error) {
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
