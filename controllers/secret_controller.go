/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/secrets"
)

const autogenerated = "autogenerated"

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Log           logr.Logger
	SecretsClient secrets.Client
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=azure.alexeldeib.xyz,resources=secrets/status,verbs=get;update;patch

func (r *SecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("secret", req.NamespacedName)

	// Fetch from Kubernetes API server
	var secret azurev1alpha1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		// dont't requeue not found
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch secret")
		return ctrl.Result{}, err
	}

	// Fetch from Azure
	remotesecret, err := r.SecretsClient.Get(ctx, &secret)
	if err != nil && !remotesecret.IsHTTPStatus(http.StatusNotFound) {
		return ctrl.Result{}, err
	}

	if remotesecret.IsHTTPStatus(http.StatusNotFound) {
		log.Info("keyvault secret not found")
		secret.Status.Exists = false
	} else {
		secret.Status.Exists = true
	}

	// Fetch target Kubernetes secret
	namespacedNamed := types.NamespacedName{
		Name:      secret.Spec.Name,
		Namespace: secret.ObjectMeta.GetNamespace(),
	}

	localsecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Spec.Name,
			Namespace: secret.ObjectMeta.GetNamespace(),
		},
	}

	if secret.Spec.LocalName != nil {
		namespacedNamed.Name = *secret.Spec.LocalName
		localsecret.Name = *secret.Spec.LocalName
	}

	localerr := r.Get(ctx, namespacedNamed, &localsecret)
	if localerr != nil && !apierrs.IsNotFound(localerr) {
		return ctrl.Result{}, localerr
	}

	if apierrs.IsNotFound(localerr) {
		log.Info("corresponding kubernetes secret not found")
		secret.Status.Available = false
	} else {
		secret.Status.Available = true
	}

	// Update our awareness of the state
	secret.Status.Generation = secret.ObjectMeta.GetGeneration()
	if err := r.Status().Update(ctx, &secret); err != nil {
		return ctrl.Result{}, err
	}

	// Handle deletion/finalizer
	if secret.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add finalizer if not present
		if !containsString(secret.ObjectMeta.Finalizers, finalizerName) {
			secret.ObjectMeta.Finalizers = append(secret.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &secret); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(secret.ObjectMeta.Finalizers, finalizerName) {
			log.Info("handling deletion of secret")
			// If the Kubernetes secret exists, delete it.
			requeue := false
			var resultErr *multierror.Error
			if secret.Status.Available {
				requeue = true
				if err = r.Delete(ctx, &localsecret); err != nil {
					log.Info("failed deletion of Kubernetes secret")
					resultErr = multierror.Append(resultErr, err)
				}
			}

			// TODO(ace): if a Keyvault exists and has the annotation "managed", delete it
			// If the Azure secret exists and has the tag "autogenerated", delete it.
			// This would naturally be done as part of Keyvault deletion if using a managed Keyvault.
			_, isAutogenerated := remotesecret.Tags[autogenerated]
			if secret.Status.Exists && isAutogenerated {
				requeue = true
				if err := r.SecretsClient.Delete(ctx, &secret); err != nil {
					log.Info("failed deletion of Keyvault secret")
					resultErr = multierror.Append(resultErr, err)
				}
			}

			if err := r.Status().Update(ctx, &secret); err != nil {
				log.Info("failed to update status after deletion")
				resultErr = multierror.Append(resultErr, err) //nolint:ineffassign
			}

			if !requeue {
				log.Info("finished deletion of secret")
				secret.ObjectMeta.Finalizers = removeString(secret.ObjectMeta.Finalizers, finalizerName)
				if err := r.Update(ctx, &secret); err != nil {
					return ctrl.Result{}, err
				}
			}
			if finalErr := resultErr.ErrorOrNil(); finalErr != nil {
				return ctrl.Result{}, finalErr
			}
			return ctrl.Result{Requeue: requeue}, nil
		}
		return ctrl.Result{}, nil
	}

	log.Info("reconciling secret")
	// If the external secret exists, sync it to Kubernetes
	if secret.Status.Exists {
		secretValue := *remotesecret.Value
		// Construct candidate owner reference
		ref, err := refFromOwner(r.Scheme, &secret)
		if err != nil {
			return ctrl.Result{}, nil
		}
		// Create or mutate
		_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &localsecret, func() error {
			// Attempt to find an owner ref matching our controller..
			owners := localsecret.GetOwnerReferences()
			add := true
			for _, owner := range owners {
				if referSameObject(owner, ref) {
					add = false
				}
			}
			// If we didn't find the owner ref, let's add it.
			if add {
				innerErr := controllerutil.SetControllerReference(&secret, &localsecret, r.Scheme) //nolint:ineffassign
				if innerErr != nil {
					return innerErr
				}
			}
			// Initialize data if necessary, and set desired key.
			if localsecret.Data == nil {
				localsecret.Data = map[string][]byte{}
			}
			// TODO(ace): key this off something else
			// TODO(ace): check for and avoid conflicts; allow/deny overwrite from user spec
			localsecret.Data[secret.Spec.Name] = []byte(secretValue)
			return nil
		})
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, errors.New("secret not found in Keyvault")

	// TODO(ace): If it doesn't exist, generate it (requires input metadata)
	// TODO(ace): create the necessary Keyvault if it doesn't exist
}

func referSameObject(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}

	return aGV == bGV && a.Kind == b.Kind && a.Name == b.Name
}

func refFromOwner(scheme *runtime.Scheme, owner runtime.Object) (metav1.OwnerReference, error) {
	gvk, err := apiutil.GVKForObject(owner, scheme)
	if err != nil {
		return metav1.OwnerReference{}, err
	}

	// Create a new ref
	ref := *metav1.NewControllerRef(owner.(metav1.Object), schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind})
	return ref, nil
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.Secret{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
