/*
Copyright 2019 Alexander Eldeib.
*/

package services

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
)

type AzureResourceGroupService interface {
	CreateOrUpdate(ctx context.Context, local *azurev1alpha1.ResourceGroup, remote resources.Group) (result resources.Group, err error)
	Get(ctx context.Context, local *azurev1alpha1.ResourceGroup) (result resources.Group, err error)
	Delete(ctx context.Context, local *azurev1alpha1.ResourceGroup, log logr.Logger) (bool, error)
}

type ResourceGroupService struct {
	autorest.Authorizer
}

func (s *ResourceGroupService) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.ResourceGroup, remote resources.Group) (result resources.Group, err error) {
	client, err := clients.NewGroupsClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return resources.Group{}, err
	}
	return client.CreateOrUpdate(ctx, local.Spec.Name, remote)
}

func (s *ResourceGroupService) Get(ctx context.Context, local *azurev1alpha1.ResourceGroup) (result resources.Group, err error) {
	client, err := clients.NewGroupsClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return resources.Group{}, err
	}
	return client.Get(ctx, local.Spec.Name)
}

func (s *ResourceGroupService) Delete(ctx context.Context, local *azurev1alpha1.ResourceGroup, log logr.Logger) (bool, error) {
	client, err := clients.NewGroupsClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return false, err
	}

	var future resources.GroupsDeleteFuture
	if local.Status.Future != nil {
		log.Info("existing future found, parsing to check remote status")
		var azureFuture azure.Future
		if err := azureFuture.UnmarshalJSON(local.Status.Future); err != nil {
			return false, err
		}
		future = resources.GroupsDeleteFuture{
			Future: azureFuture,
		}
	} else {
		log.Info("no existing future found, beginning deletion")
		deleteFuture, err := client.Delete(ctx, local.Spec.Name)
		if err != nil {
			if res := deleteFuture.Response(); res != nil && res.StatusCode == http.StatusNotFound {
				// Not found is successful delete on a resource.
				return true, nil
			}
			return false, err
		}
		future = deleteFuture
	}

	log.Info("checking deletion status")
	done, err := future.DoneWithContext(ctx, &client)
	if err != nil {
		return false, err
	}

	if done {
		local.Status.Future = nil
	} else {
		local.Status.Future, err = future.MarshalJSON()
	}

	return done, err
}
