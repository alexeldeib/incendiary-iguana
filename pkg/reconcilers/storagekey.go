/*
Copyright 2019 Alexander Eldeib.
*/

package reconcilers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/services"
)

type StorageKeyReconciler struct {
	Service    *services.StorageService
	Kubeclient ctrl.Client
	*runtime.Scheme
}

func (r *StorageKeyReconciler) listKeys(ctx context.Context, local *azurev1alpha1.StorageKey) (map[string][]byte, error) {
	keys, err := r.Service.ListKeys(ctx, r.accountFromKey(local))
	if err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	// TODO(ace): 10/22/2019 - I still think this is safe at runtime? looks insane.
	if local.Spec.PrimaryKey != nil {
		result[*local.Spec.PrimaryKey] = []byte(*(*keys.Keys)[0].Value)
	}
	// TODO(ace): sovereign clouds, suffix should come from env/stored in reconciler
	if local.Spec.PrimaryConnectionString != nil {
		connectionString := fmt.Sprintf("DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net", local.Spec.Name, *(*keys.Keys)[0].Value)
		result[*local.Spec.PrimaryConnectionString] = []byte(connectionString)
	}
	return result, nil
}

func (r *StorageKeyReconciler) Ensure(ctx context.Context, obj runtime.Object) error {
	local, err := r.convert(obj)
	if err != nil {
		return err
	}

	if local.Spec.TargetSecret == nil {
		return nil
	}

	keys, err := r.listKeys(ctx, local)
	if err != nil {
		return err
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *local.Spec.TargetSecret,
			Namespace: local.ObjectMeta.Namespace,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Kubeclient, targetSecret, func() error {
		if targetSecret.Data == nil {
			targetSecret.Data = map[string][]byte{}
		}
		for key, val := range keys {
			targetSecret.Data[key] = []byte(val)
		}
		if local.ObjectMeta.UID != "" {
			if ownerErr := controllerutil.SetControllerReference(local, targetSecret, r.Scheme); ownerErr != nil {
				return ownerErr
			}
		}
		return nil
	})

	return err
}

// Delete handles deletion of a SQL server.
func (r *StorageKeyReconciler) Delete(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	local, err := r.convert(obj)
	if err != nil {
		return err
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      local.Spec.Name,
			Namespace: local.ObjectMeta.Namespace,
		},
	}

	err = r.Kubeclient.Delete(ctx, targetSecret)
	return client.IgnoreNotFound(err)
}

func (r *StorageKeyReconciler) convert(obj runtime.Object) (*azurev1alpha1.StorageKey, error) {
	local, ok := obj.(*azurev1alpha1.StorageKey)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}

func (r *StorageKeyReconciler) accountFromKey(key *azurev1alpha1.StorageKey) *azurev1alpha1.StorageAccount {
	return &azurev1alpha1.StorageAccount{
		Spec: azurev1alpha1.StorageAccountSpec{
			SubscriptionID: key.Spec.SubscriptionID,
			ResourceGroup:  key.Spec.ResourceGroup,
			Name:           key.Spec.Name,
		},
	}
}
