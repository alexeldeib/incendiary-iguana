/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/sanity-io/litter"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/secrets"
)

var BadHostRegex = regexp.MustCompile(`StatusCode=([0-9]{0,3})`)

// SecretBundleReconciler reconciles a Secret object
type SecretBundleReconciler struct {
	client.Client
	Log           logr.Logger
	SecretsClient secrets.Client
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=secretbundles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=secretbundles/status,verbs=get;update;patch

func (r *SecretBundleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("secretBundle", req.NamespacedName)

	// Fetch from Kubernetes API server
	var secretBundle azurev1alpha1.SecretBundle
	if err := r.Get(ctx, req.NamespacedName, &secretBundle); err != nil {
		log.Info("unable to fetch secret")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// UPDATE STATUS
	remoteSecretsStatus := azurev1alpha1.SecretBundleStatus{
		Generation: secretBundle.ObjectMeta.GetGeneration(),
		Desired:    len(secretBundle.Spec.Secrets),
		Secrets:    map[string]azurev1alpha1.SingleSecretStatus{},
	}

	newSecretData := map[string][]byte{}

	// Iterate desired secrets and sync their status.
	available := 0
	for _, item := range secretBundle.Spec.Secrets {
		arg := &azurev1alpha1.Secret{
			Spec: azurev1alpha1.SecretSpec{
				Name:  item.Name,
				Vault: item.Vault,
			},
		}
		remoteSecret, err := r.SecretsClient.Get(ctx, arg)
		if err != nil && !remoteSecret.HasHTTPStatus(http.StatusNotFound) {
			// TODO(ace): BLOCKER should handle this more gracefully
			matches := BadHostRegex.FindSubmatch([]byte(err.Error()))
			if matches == nil {
				return ctrl.Result{}, err
			}
			parts := strings.Split(string(matches[0]), "=")
			if len(parts) < 2 || parts[1] != "0" {
				litter.Dump(parts[1])
				return ctrl.Result{}, err
			}
		}
		localName := item.Name
		if item.LocalName != nil {
			localName = *item.LocalName
		}
		if !remoteSecret.IsHTTPStatus(http.StatusOK) {
			log.Info("keyvault secret not found")
			remoteSecretsStatus.Secrets[localName] = azurev1alpha1.SingleSecretStatus{
				Exists:    false,
				Available: false,
			}
		} else {
			available = available + 1
			newSecretData[localName] = []byte(*remoteSecret.Value)
			remoteSecretsStatus.Secrets[localName] = azurev1alpha1.SingleSecretStatus{
				Exists:    true,
				Available: false,
			}
		}
	}

	// Fetch corresponding Kubernetes secret
	namespacedNamed := types.NamespacedName{
		Name:      secretBundle.Spec.Name,
		Namespace: secretBundle.ObjectMeta.GetNamespace(),
	}

	// Construct the desired secret object
	localSecretBundle := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedNamed.Name,
			Namespace: namespacedNamed.Namespace,
		},
		Data: map[string][]byte{},
	}

	// Try to get the existing object
	localerr := r.Get(ctx, namespacedNamed, &localSecretBundle)
	if client.IgnoreNotFound(localerr) != nil {
		return ctrl.Result{}, localerr
	}

	// Handle local missing secret
	if apierrs.IsNotFound(localerr) {
		log.Info("corresponding kubernetes secret not found")
		remoteSecretsStatus.Ready = 0
	} else {
		remoteSecretsStatus.Ready = len(localSecretBundle.Data)
	}

	// Update availability status from secret data
	remoteSecretsStatus.Ready = len(localSecretBundle.Data)
	remoteSecretsStatus.Available = len(newSecretData)
	for name := range localSecretBundle.Data {
		if val, ok := remoteSecretsStatus.Secrets[name]; ok {
			val.Available = true
			remoteSecretsStatus.Secrets[name] = val
		} else {
			newSecretData[name] = localSecretBundle.Data[name]
			remoteSecretsStatus.Secrets[name] = azurev1alpha1.SingleSecretStatus{
				Exists:    false,
				Available: true,
			}
		}
	}

	// Update our awareness of the state
	secretBundle.Status = remoteSecretsStatus
	if err := r.Status().Update(ctx, &secretBundle); err != nil {
		return ctrl.Result{}, err
	}

	if !secretBundle.ObjectMeta.DeletionTimestamp.IsZero() {
		if contains(secretBundle.ObjectMeta.Finalizers, finalizerName) {
			log.Info("handling deletion of secret bundle")
			if err := r.Delete(ctx, &localSecretBundle); client.IgnoreNotFound(err) != nil {
				log.Info("failed deletion of Kubernetes secret")
				return ctrl.Result{}, err
			}
			if err := r.Status().Update(ctx, &secretBundle); err != nil {
				log.Info("failed to update status after deletion")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, RemoveFinalizerAndUpdate(ctx, r.Client, finalizerName, &secretBundle)
		}
	}

	log.Info("reconciling secret")
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, &localSecretBundle, func() error {
		log.Info("mutating secret bundle")
		innerErr := controllerutil.SetControllerReference(&secretBundle, &localSecretBundle, r.Scheme)
		if innerErr != nil {
			return innerErr
		}
		log.Info("setting secret map data")
		localSecretBundle.Data = newSecretData
		return nil
	})

	return ctrl.Result{}, err
}

func (r *SecretBundleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.SecretBundle{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
