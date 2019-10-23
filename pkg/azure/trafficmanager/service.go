/*
Copyright 2019 Alexander Eldeib.
*/

package trafficmanager

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/trafficmanager/mgmt/trafficmanager"
	"github.com/Azure/go-autorest/autorest"
	"github.com/go-logr/logr"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
)

type TrafficManagerService struct {
	autorest.Authorizer
}

func (s *TrafficManagerService) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.TrafficManager, remote trafficmanager.Profile) error {
	client, err := clients.NewProfilesClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return err
	}
	_, err = client.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, remote)
	return err
}

func (s *TrafficManagerService) Get(ctx context.Context, local *azurev1alpha1.TrafficManager) (trafficmanager.Profile, error) {
	client, err := clients.NewProfilesClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return trafficmanager.Profile{}, err
	}
	return client.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

func (s *TrafficManagerService) Delete(ctx context.Context, local *azurev1alpha1.TrafficManager, log logr.Logger) error {
	client, err := clients.NewProfilesClient(local.Spec.SubscriptionID, s.Authorizer)
	if err != nil {
		return err
	}

	_, err = client.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	return err
}
