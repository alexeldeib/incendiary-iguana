/*
Copyright 2019 Alexander Eldeib.
*/

package trafficmanagers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/trafficmanager/mgmt/2018-04-01/trafficmanager"
	"github.com/Azure/go-autorest/autorest/to"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

type Client struct {
	factory  factoryFunc
	internal trafficmanager.ProfilesClient
	config   *config.Config
}

type factoryFunc func(subscriptionID string) trafficmanager.ProfilesClient

// New returns a new client able to authenticate to multiple Azure subscriptions using the provided configuration.
func New(configuration *config.Config) *Client {
	return NewWithFactory(configuration, trafficmanager.NewProfilesClient)
}

// NewWithFactory returns an interface which can authorize the configured client to many subscriptions.
// It uses the factory argument to instantiate new clients for a specific subscription.
// This can be used to stub Azure client for testing.
func NewWithFactory(configuration *config.Config, factory factoryFunc) *Client {
	return &Client{
		config:  configuration,
		factory: factory,
	}
}

// ForSubscription authorizes the client for a given subscription
func (c *Client) ForSubscription(subID string) error {
	c.internal = c.factory(subID)
	return c.config.AuthorizeClient(&c.internal.Client)
}

// Ensure creates or updates a virtual network in an idempotent manner and sets its provisioning state.
func (c *Client) Ensure(ctx context.Context, local *azurev1alpha1.TrafficManager) (bool, error) {
	spec := trafficmanager.Profile{
		ProfileProperties: &trafficmanager.ProfileProperties{
			ProfileStatus:        trafficmanager.ProfileStatus(local.Spec.ProfileStatus),
			TrafficRoutingMethod: trafficmanager.TrafficRoutingMethod(local.Spec.TrafficRoutingMethod),
			MonitorConfig: &trafficmanager.MonitorConfig{
				Protocol:                  trafficmanager.MonitorProtocol(local.Spec.MonitorConfig.Protocol),
				Port:                      local.Spec.MonitorConfig.Port,
				Path:                      to.StringPtr("/healthz"),
				IntervalInSeconds:         local.Spec.MonitorConfig.IntervalInSeconds,
				TimeoutInSeconds:          local.Spec.MonitorConfig.TimeoutInSeconds,
				ToleratedNumberOfFailures: local.Spec.MonitorConfig.ToleratedNumberOfFailures,
				CustomHeaders:             &[]trafficmanager.MonitorConfigCustomHeadersItem{},
				ExpectedStatusCodeRanges:  &[]trafficmanager.MonitorConfigExpectedStatusCodeRangesItem{},
			},
			Endpoints: &[]trafficmanager.Endpoint{},
			DNSConfig: &trafficmanager.DNSConfig{
				RelativeName: local.Spec.DNSConfig.RelativeName,
				Fqdn:         to.StringPtr(fmt.Sprintf("%s.trafficmanager.net", *local.Spec.DNSConfig.RelativeName)),
				TTL:          local.Spec.DNSConfig.TTL,
			},
		},
		Location: to.StringPtr("global"),
	}

	if local.Spec.MonitorConfig.Path != nil {
		spec.ProfileProperties.MonitorConfig.Path = local.Spec.MonitorConfig.Path
	}

	if local.Spec.MonitorConfig.CustomHeaders != nil {
		for _, header := range *local.Spec.MonitorConfig.CustomHeaders {
			new := trafficmanager.MonitorConfigCustomHeadersItem{
				Name:  header.Name,
				Value: header.Value,
			}
			*spec.ProfileProperties.MonitorConfig.CustomHeaders = append(*spec.ProfileProperties.MonitorConfig.CustomHeaders, new)
		}
	}

	if local.Spec.MonitorConfig.CustomHeaders != nil {
		for _, spread := range *local.Spec.MonitorConfig.ExpectedStatusCodeRanges {
			new := trafficmanager.MonitorConfigExpectedStatusCodeRangesItem{
				Min: spread.Min,
				Max: spread.Max,
			}
			*spec.ProfileProperties.MonitorConfig.ExpectedStatusCodeRanges = append(*spec.ProfileProperties.MonitorConfig.ExpectedStatusCodeRanges, new)
		}
	}

	if local.Spec.Endpoints != nil {
		for _, ep := range *local.Spec.Endpoints {
			endpointSpec := trafficmanager.Endpoint{
				Name: to.StringPtr(ep.Name),
				Type: to.StringPtr("Microsoft.Network/trafficManagerProfiles/externalEndpoints"),
				EndpointProperties: &trafficmanager.EndpointProperties{
					Target:           ep.Properties.Target,
					Weight:           ep.Properties.Weight,
					Priority:         to.Int64Ptr(ep.Properties.Priority),
					EndpointLocation: to.StringPtr(ep.Properties.EndpointLocation),
				},
			}
			if ep.Properties.CustomHeaders != nil {
				endpointSpec.CustomHeaders = &[]trafficmanager.EndpointPropertiesCustomHeadersItem{}
				for _, header := range *ep.Properties.CustomHeaders {
					item := trafficmanager.EndpointPropertiesCustomHeadersItem{
						Name:  header.Name,
						Value: header.Value,
					}
					*endpointSpec.CustomHeaders = append(*endpointSpec.CustomHeaders, item)
				}
			}
			*spec.ProfileProperties.Endpoints = append(*spec.ProfileProperties.Endpoints, endpointSpec)
		}
	}

	if _, err := c.internal.CreateOrUpdate(ctx, local.Spec.ResourceGroup, local.Spec.Name, spec); err != nil {
		return false, err
	}

	if _, err := c.SetStatus(ctx, local); err != nil {
		return false, err
	}

	return c.Done(ctx, local), nil
}

// Delete handles deletion of a virtual network.
func (c *Client) Delete(ctx context.Context, local *azurev1alpha1.TrafficManager) error {
	_, err := c.internal.Delete(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	return err
}

// SetStatus sets the status subresource fields of the CRD reflecting the state of the object in Azure.
func (c *Client) SetStatus(ctx context.Context, local *azurev1alpha1.TrafficManager) (bool, error) {
	remote, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	// Care about 400 and 5xx, not 404.
	found := !remote.IsHTTPStatus(http.StatusNotFound)
	if err != nil && !remote.HasHTTPStatus(http.StatusNotFound, http.StatusConflict) {
		return found, err
	}

	local.Status.ID = remote.ID
	if remote.ProfileProperties != nil {
		local.Status.FQDN = remote.ProfileProperties.DNSConfig.Fqdn
		local.Status.ProfileMonitorStatus = string(remote.ProfileProperties.MonitorConfig.ProfileMonitorStatus)
	}
	return found, nil
}

// Done checks the current state of the CRD against the desired end state.
func (c *Client) Done(ctx context.Context, local *azurev1alpha1.TrafficManager) bool {
	// TODO(ace): make this check individual endpoints? what about ICMs?
	return local.Status.ProfileMonitorStatus == "Online"
}

// Get returns a virtual network.
func (c *Client) Get(ctx context.Context, local *azurev1alpha1.TrafficManager) (trafficmanager.Profile, error) {
	return c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
}

// remove?
// GetProfileStatus returns the status of an entire Azure TM.
func (c *Client) GetProfileStatus(ctx context.Context, local *azurev1alpha1.TrafficManager) (string, error) {
	res, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		return "", err
	}
	return string(res.ProfileProperties.MonitorConfig.ProfileMonitorStatus), nil
}

// remove?
// GetEndpointStatus returns the status of one endpoint within an Azure Traffic Manager.
func (c *Client) GetEndpointStatus(ctx context.Context, local *azurev1alpha1.TrafficManager, name string) (string, error) {
	profile, err := c.internal.Get(ctx, local.Spec.ResourceGroup, local.Spec.Name)
	if err != nil {
		return "", err
	}
	for _, ep := range *profile.ProfileProperties.Endpoints {
		if *ep.Name == name {
			return string(ep.EndpointMonitorStatus), nil
		}
	}
	return "", errors.New("endpoint not found in current tm configuration")
}
