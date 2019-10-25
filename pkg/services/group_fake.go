/*
Copyright 2019 Alexander Eldeib.
*/

package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/go-logr/logr"
)

type FakeResourceGroupService struct {
	State  map[string]string
	Exists map[string]bool
	Start  time.Time
}

func (f *FakeResourceGroupService) CreateOrUpdate(ctx context.Context, local *azurev1alpha1.ResourceGroup, remote resources.Group) (resources.Group, error) {
	f.State[local.Spec.Name] = "Succeeded"
	f.Exists[local.Spec.Name] = true
	return f.getResult(local), nil
}

func (f *FakeResourceGroupService) Get(ctx context.Context, local *azurev1alpha1.ResourceGroup) (result resources.Group, err error) {
	if f.Exists[local.Spec.Name] {
		return f.getResult(local), nil
	}
	return f.getResult(local), errors.New("not found error")
}

func (f *FakeResourceGroupService) Delete(ctx context.Context, local *azurev1alpha1.ResourceGroup, log logr.Logger) (bool, error) {
	f.State[local.Spec.Name] = "Deleting"
	time.AfterFunc(10*time.Second, func() {
		f.State[local.Spec.Name] = "Deleted"
		f.Exists[local.Spec.Name] = false
	})
	return !f.Exists[local.Spec.Name], nil
}

func (f *FakeResourceGroupService) getResult(local *azurev1alpha1.ResourceGroup) resources.Group {
	return resources.Group{
		ID:       to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", local.Spec.SubscriptionID, local.Spec.Name)),
		Location: &local.Spec.Location,
		Name:     &local.Spec.Name,
		Properties: &resources.GroupProperties{
			ProvisioningState: to.StringPtr(f.State[local.Spec.Name]),
		},
		Response: autorest.Response{
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		},
	}
}
