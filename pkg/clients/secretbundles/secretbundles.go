/*
Copyright 2019 Alexander Eldeib.
*/

package secretbundles

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"software.sslmate.com/src/go-pkcs12"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/redis"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/servicebus"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/tlssecrets"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	internal         keyvault.BaseClient
	configuration    *config.Config
	kubeclient       *ctrl.Client
	scheme           *runtime.Scheme
	redisClient      *redis.Client
	servicebusClient *servicebus.Client
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
	return &Client{
		internal:      kvclient,
		kubeclient:    kubeclient,
		scheme:        scheme,
		configuration: configuration,
	}, nil
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(ctx context.Context, obj runtime.Object) error {
	return nil
}

// Ensure takes a spec corresponding to one Azure KV secret. It syncs that secret into Kubernetes, remapping the name if necessary.
func (c *Client) Ensure(ctx context.Context, obj runtime.Object) error {
	// TODO(ace): cloud-sensitive
	secret, err := c.convert(obj)
	if err != nil {
		return err
	}
	secrets := map[string][]byte{}
	for name, item := range secret.Spec.Secrets {
		// TODO(ace): more graceful error handling?
		// parallelize and collect?
		vault := fmt.Sprintf("https://%s.%s", item.Vault, azure.PublicCloud.KeyVaultDNSSuffix)
		if item.Kind == nil {
			bundle, err := c.internal.GetSecret(ctx, vault, item.Name, "")
			if err != nil {
				return err
			}
			secrets[name] = []byte(*bundle.Value)
			continue
		}

		switch *item.Kind {
		case "sha":
			// Shortcircuit SHA handling because it uses a different client
			cert, err := c.internal.GetCertificate(ctx, vault, item.Name, "")
			if err != nil {
				return err
			}
			out, err := formatSHA(*cert.X509Thumbprint)
			if err != nil {
				return err
			}
			secrets[name] = out
		default:
			bundle, err := c.internal.GetSecret(ctx, vault, item.Name, "")
			if err != nil {
				return err
			}
			output, err := format(item.Kind, *bundle.Value, item.Reverse)
			if err != nil {
				return err
			}
			secrets[name] = output
		}
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
		for key, val := range secrets {
			local.Data[key] = val
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
func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	secret, err := c.convert(obj)
	if err != nil {
		return err
	}
	local := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Spec.Name,
			Namespace: secret.ObjectMeta.Namespace,
		},
	}
	return client.IgnoreNotFound((*c.kubeclient).Delete(ctx, local))
}

func format(format *string, secret string, reverse bool) ([]byte, error) {
	// PKCS12/pfx data is what we have and what we want, bail out successfully
	if format == nil {
		return []byte(secret), nil
	}
	switch kind := *format; kind {
	case "pkcs8":
		return formatPKCS8(secret)
	case "rsa":
		return formatRSA(secret)
	case "x509":
		return formatX509(secret, reverse)
	default:
		return nil, errors.New("failed to find secret format")
	}
}

func parsePKCS12(secret string) (privateKey interface{}, certificate *x509.Certificate, caCerts []*x509.Certificate, err error) {
	p12, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "err decoding base64 to p12")
	}
	return pkcs12.DecodeChain(p12, "")
}

func formatSHA(thumbprint string) ([]byte, error) {
	src, err := base64.RawURLEncoding.DecodeString(thumbprint)
	if err != nil {
		return nil, err
	}
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	dst = []byte(strings.ToUpper(string(dst)))
	return dst, nil
}

func formatPKCS8(secret string) ([]byte, error) {
	pfxKey, _, _, err := parsePKCS12(secret)
	if err != nil {
		return nil, err
	}

	keyX509, err := x509.MarshalPKCS8PrivateKey(pfxKey)
	if err != nil {
		return nil, err
	}
	keyBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyX509,
	}
	var keyPEM bytes.Buffer
	if err := pem.Encode(&keyPEM, keyBlock); err != nil {
		return nil, err
	}
	return keyPEM.Bytes(), nil
}

func formatRSA(secret string) ([]byte, error) {
	pfxKey, _, _, err := parsePKCS12(secret)
	if err != nil {
		return nil, err
	}

	keyX509 := x509.MarshalPKCS1PrivateKey(pfxKey.(*rsa.PrivateKey))
	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyX509,
	}

	var keyPEM bytes.Buffer
	if err := pem.Encode(&keyPEM, keyBlock); err != nil {
		return nil, err
	}
	return keyPEM.Bytes(), nil
}

func formatX509(secret string, reverse bool) ([]byte, error) {
	_, pfxCert, caCerts, err := parsePKCS12(secret)
	if err != nil {
		return nil, err
	}

	if reverse {
		pfxCert, caCerts[len(caCerts)-1] = caCerts[len(caCerts)-1], pfxCert
	}

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: pfxCert.Raw,
	}

	var certPEM bytes.Buffer
	if err := pem.Encode(&certPEM, certBlock); err != nil {
		return nil, err
	}

	// append certificates to create chain
	output := fmt.Sprintf("%s\n%s\n%s", tlssecrets.GenerateSubject(pfxCert), tlssecrets.GenerateIssuer(pfxCert), certPEM.String())
	caCertString := ""
	for _, cert := range caCerts {
		certBlock = &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		var certPEM bytes.Buffer
		if err := pem.Encode(&certPEM, certBlock); err != nil {
			return nil, err
		}
		caCertString = fmt.Sprintf("%s\n%s\n%s\n%s", caCertString, tlssecrets.GenerateSubject(cert), tlssecrets.GenerateIssuer(cert), certPEM.String())
	}
	output = fmt.Sprintf("%s\n%s", output, caCertString)

	return []byte(output), nil
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.SecretBundle, error) {
	local, ok := obj.(*azurev1alpha1.SecretBundle)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
