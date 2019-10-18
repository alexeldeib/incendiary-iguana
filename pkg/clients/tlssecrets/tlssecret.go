/*
Copyright 2019 Alexander Eldeib.
*/

package tlssecrets

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

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
	pkcs12 "software.sslmate.com/src/go-pkcs12"

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
	// noop for keyvault secrets and kubeclients
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

	p12, err := base64.StdEncoding.DecodeString(*bundle.Value)
	if err != nil {
		return errors.Wrapf(err, "err decoding base64 to p12")
	}
	pfxKey, pfxCert, caCerts, err := pkcs12.DecodeChain(p12, "")
	if err != nil {
		return err
	}
	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: pfxCert.Raw,
	}
	var certPEM bytes.Buffer
	pem.Encode(&certPEM, certBlock)
	output := fmt.Sprintf("%s\n%s\n%s", GenerateSubject(pfxCert), GenerateIssuer(pfxCert), certPEM.String())
	caCertString := ""
	// Fix cert chain order (reverse them and fix headers)
	for _, cert := range caCerts {
		certBlock = &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		var certPEM bytes.Buffer
		pem.Encode(&certPEM, certBlock)
		caCertString = fmt.Sprintf("%s\n%s\n%s\n%s", caCertString, GenerateSubject(cert), GenerateIssuer(cert), certPEM.String())
	}
	output = fmt.Sprintf("%s\n%s", output, caCertString)
	keyX509 := x509.MarshalPKCS1PrivateKey(pfxKey.(*rsa.PrivateKey))
	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyX509,
	}
	var keyPEM bytes.Buffer
	if err := pem.Encode(&keyPEM, keyBlock); err != nil {
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
			innerErr := controllerutil.SetControllerReference(secret, local, c.scheme)
			if innerErr != nil {
				return innerErr
			}
		}
		if local.Data == nil {
			local.Data = map[string][]byte{}
		}
		local.Data["tls.crt"] = []byte(output)
		local.Data["tls.key"] = keyPEM.Bytes()
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

func GenerateSubject(cert *x509.Certificate) string {
	subject := "subject="
	if cert.Subject.Country != nil {
		subject = fmt.Sprintf("%s/C=%s", subject, cert.Subject.Country[0])
	}
	if cert.Subject.Province != nil {
		subject = fmt.Sprintf("%s/ST=%s", subject, cert.Subject.Province[0])
	}
	if cert.Subject.Locality != nil {
		subject = fmt.Sprintf("%s/L=%s", subject, cert.Subject.Locality[0])
	}
	if cert.Subject.Organization != nil {
		subject = fmt.Sprintf("%s/O=%s", subject, cert.Subject.Organization[0])
	}
	if cert.Subject.OrganizationalUnit != nil {
		subject = fmt.Sprintf("%s/OU=%s", subject, cert.Subject.OrganizationalUnit[0])
	}
	return fmt.Sprintf("%s/CN=%s", subject, cert.Subject.CommonName)
}

func GenerateIssuer(cert *x509.Certificate) string {
	issuer := "issuer="
	if cert.Issuer.Country != nil {
		issuer = fmt.Sprintf("%s/C=%s", issuer, cert.Issuer.Country[0])
	}
	if cert.Issuer.Province != nil {
		issuer = fmt.Sprintf("%s/ST=%s", issuer, cert.Issuer.Province[0])
	}
	if cert.Issuer.Locality != nil {
		issuer = fmt.Sprintf("%s/L=%s", issuer, cert.Issuer.Locality[0])
	}
	if cert.Issuer.Organization != nil {
		issuer = fmt.Sprintf("%s/O=%s", issuer, cert.Issuer.Organization[0])
	}
	if cert.Issuer.OrganizationalUnit != nil {
		issuer = fmt.Sprintf("%s/OU=%s", issuer, cert.Issuer.OrganizationalUnit[0])
	}
	return fmt.Sprintf("%s/CN=%s", issuer, cert.Issuer.CommonName)
}

func (c *Client) convert(obj runtime.Object) (*azurev1alpha1.TLSSecret, error) {
	local, ok := obj.(*azurev1alpha1.TLSSecret)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
