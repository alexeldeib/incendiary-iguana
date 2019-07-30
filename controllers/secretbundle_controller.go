/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
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
	// FETCH
	var secretBundle azurev1alpha1.SecretBundle
	if err := r.Get(ctx, req.NamespacedName, &secretBundle); err != nil {
		// dont't requeue not found
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch secret")
		return ctrl.Result{}, err
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
		if err != nil && !remoteSecret.IsHTTPStatus(http.StatusNotFound) {
			// TODO(ace): BLOCKER should handle this more gracefully
			return ctrl.Result{}, err
		}
		if remoteSecret.IsHTTPStatus(http.StatusNotFound) {
			log.Info("keyvault secret not found")
			remoteSecretsStatus.Secrets[item.Name] = azurev1alpha1.SingleSecretStatus{
				Exists:    false,
				Available: false,
			}
		} else {
			available = available + 1
			if item.LocalName != nil {
				newSecretData[*item.LocalName] = []byte(*remoteSecret.Value)
			} else {
				newSecretData[item.Name] = []byte(*remoteSecret.Value)
			}
			remoteSecretsStatus.Secrets[item.Name] = azurev1alpha1.SingleSecretStatus{
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

	// Construct the Kubernetes secret bundle
	localSecretBundle := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedNamed.Name,
			Namespace: namespacedNamed.Namespace,
		},
		Data: map[string][]byte{},
	}

	// Try to get the Kubernetes bundle
	localerr := r.Get(ctx, namespacedNamed, &localSecretBundle)
	if localerr != nil && !apierrs.IsNotFound(localerr) {
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

	// ADD FINALIZER (util func)
	// Handle deletion/finalizer
	if secretBundle.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add finalizer if not present
		if !containsString(secretBundle.ObjectMeta.Finalizers, finalizerName) {
			secretBundle.ObjectMeta.Finalizers = append(secretBundle.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &secretBundle); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(secretBundle.ObjectMeta.Finalizers, finalizerName) {
			// DELETE
			log.Info("handling deletion of secret bundle")
			requeue := false
			var resultErr *multierror.Error

			// If the Kubernetes secret exists, delete it.
			if !apierrs.IsNotFound(localerr) {
				requeue = true
				if err := r.Delete(ctx, &localSecretBundle); err != nil {
					log.Info("failed deletion of Kubernetes secret")
					resultErr = multierror.Append(resultErr, err)
				}
			}

			// Sync our status post deletion
			if err := r.Status().Update(ctx, &secretBundle); err != nil {
				log.Info("failed to update status after deletion")
				resultErr = multierror.Append(resultErr, err) //nolint:ineffassign
			}

			// Cleanup finalizer
			if !requeue {
				log.Info("finished deletion of secret")
				secretBundle.ObjectMeta.Finalizers = removeString(secretBundle.ObjectMeta.Finalizers, finalizerName)
				if err := r.Update(ctx, &secretBundle); err != nil {
					resultErr = multierror.Append(resultErr, err)
				}
			}
			return ctrl.Result{Requeue: requeue}, resultErr.ErrorOrNil()
		}
		return ctrl.Result{}, nil
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
