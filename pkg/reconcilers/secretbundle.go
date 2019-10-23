/*
Copyright 2019 Alexander Eldeib.
*/

package reconcilers

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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"software.sslmate.com/src/go-pkcs12"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services"
	"github.com/alexeldeib/incendiary-iguana/pkg/tlsutil"
)

type SecretBundleReconciler struct {
	Service    *services.SecretService
	Kubeclient ctrl.Client
	Scheme     *runtime.Scheme
}

type secretFormatterFn func(string) ([]byte, error)

var (
	_ (secretFormatterFn) = formatSHA
	_ (secretFormatterFn) = formatPKCS8
	_ (secretFormatterFn) = formatRSA
	_ (secretFormatterFn) = formatX509Default
	_ (secretFormatterFn) = formatX509Reverse
)

// Ensure takes a spec corresponding to several Azure KV secrets. It syncs them into Kubernetes, remapping the names as necessary.
func (r *SecretBundleReconciler) Ensure(ctx context.Context, obj runtime.Object) error {
	secret, err := r.convert(obj)
	if err != nil {
		return err
	}

	secrets := map[string][]byte{}
	for name, item := range secret.Spec.Secrets {
		out, err := downloadAndFormat(ctx, r.Service, item)
		if err != nil {
			return errors.Wrapf(err, "failed to download and format secret %s", name)
		}
		secrets[name] = out
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
		for key, val := range secrets {
			local.Data[key] = val
		}
		return nil
	})

	return err
}

// Delete deletes a secret from Keyvault.
func (r *SecretBundleReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
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

func downloadAndFormat(ctx context.Context, service *services.SecretService, item azurev1alpha1.SecretIdentifier) ([]byte, error) {
	if item.Kind == nil {
		content, err := download(ctx, service, item.Vault, item.Name)
		if err != nil {
			return nil, err
		}
		return []byte(content), nil
	}

	if *item.Kind == "sha" {
		content, err := downloadCertificateSHA(ctx, service, item.Vault, item.Name)
		if err != nil {
			return nil, err
		}
		output, err := formatSHA(content)
		if err != nil {
			return nil, err
		}
		return []byte(output), err
	}

	content, err := download(ctx, service, item.Vault, item.Name)
	if err != nil {
		return nil, err
	}

	switch format := *item.Kind; format {
	case "pkcs8":
		output, err := formatPKCS8(content)
		if err != nil {
			return nil, err
		}
		return []byte(output), nil
	case "rsa":
		output, err := formatPKCS8(content)
		if err != nil {
			return nil, err
		}
		return []byte(output), nil
	case "x509":
		switch item.Reverse {
		case true:
			output, err := formatX509Reverse(content)
			if err != nil {
				return nil, err
			}
			return []byte(output), nil
		default:
			output, err := formatX509Default(content)
			if err != nil {
				return nil, err
			}
			return []byte(output), nil
		}
	}
	return nil, errors.New("failed to find secret format")
}

func download(ctx context.Context, service *services.SecretService, vault, name string) (string, error) {
	secret, err := service.Get(ctx, vault, name)
	if err != nil {
		return "", err
	}
	if secret.Value == nil {
		return "", errors.New("expected non-nil secret, got nil")
	}
	return *secret.Value, nil
}

func downloadCertificateSHA(ctx context.Context, service *services.SecretService, vault, name string) (string, error) {
	cert, err := service.GetCertificate(ctx, vault, name)
	if err != nil {
		return "", err
	}
	return *cert.X509Thumbprint, nil
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

func formatX509Default(secret string) ([]byte, error) {
	_, pfxCert, caCerts, err := parsePKCS12(secret)
	if err != nil {
		return nil, err
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
	output := fmt.Sprintf("%s\n%s\n%s", tlsutil.GenerateSubject(pfxCert), tlsutil.GenerateIssuer(pfxCert), certPEM.String())
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
		caCertString = fmt.Sprintf("%s\n%s\n%s\n%s", caCertString, tlsutil.GenerateSubject(cert), tlsutil.GenerateIssuer(cert), certPEM.String())
	}
	output = fmt.Sprintf("%s\n%s", output, caCertString)

	return []byte(output), nil
}

func formatX509Reverse(secret string) ([]byte, error) {
	_, pfxCert, caCerts, err := parsePKCS12(secret)
	if err != nil {
		return nil, err
	}

	// Flip leaf and root.
	pfxCert, caCerts[len(caCerts)-1] = caCerts[len(caCerts)-1], pfxCert

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: pfxCert.Raw,
	}

	var certPEM bytes.Buffer
	if err := pem.Encode(&certPEM, certBlock); err != nil {
		return nil, err
	}

	// append certificates to create chain
	output := fmt.Sprintf("%s\n%s\n%s", tlsutil.GenerateSubject(pfxCert), tlsutil.GenerateIssuer(pfxCert), certPEM.String())
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
		caCertString = fmt.Sprintf("%s\n%s\n%s\n%s", caCertString, tlsutil.GenerateSubject(cert), tlsutil.GenerateIssuer(cert), certPEM.String())
	}
	output = fmt.Sprintf("%s\n%s", output, caCertString)

	return []byte(output), nil
}

func (r *SecretBundleReconciler) convert(obj runtime.Object) (*azurev1alpha1.SecretBundle, error) {
	local, ok := obj.(*azurev1alpha1.SecretBundle)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
