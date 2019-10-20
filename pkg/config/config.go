package config

import (
	"errors"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Config holds the configured useragent, environment, and authentication
// credentials. The environment will notably be used to identify what resource
// to request tokens for e.g. in sovereign clouds.
type Config struct {
	userAgent string
	env       *azure.Environment
	app       string
	key       string
	tenant    string
}

type Option func(*Config)

// New stores the environment and authentication configuration.
// It uses this information to produces authorizers for various resources.
func New(opts ...Option) (*Config, error) {
	var err error
	var settings auth.EnvironmentSettings

	if settings, err = auth.GetSettingsFromEnvironment(); err != nil {
		return nil, err
	}

	c := &Config{
		userAgent: "azauth",
		env:       &settings.Environment,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// UserAgent sets the user agent on Azure SDK clients.
func UserAgent(userAgent string) Option {
	return func(c *Config) {
		c.userAgent = userAgent
	}
}

// App sets the AAD application to authenticate with.
func App(app string) Option {
	return func(c *Config) {
		c.app = app
	}
}

// Key sets the client secret for the AAD application used in authentication.
func Key(key string) Option {
	return func(c *Config) {
		c.key = key
	}
}

// Tenant sets the tenant ID for authentication and token acquisition.
func Tenant(tenant string) Option {
	return func(c *Config) {
		c.tenant = tenant
	}
}

// AuthorizeClientForResource fetches an authorizer from the environment for a given resources and injects it to a client.
func (c *Config) AuthorizeClientForResource(client *autorest.Client, resource string) (err error) {
	authorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(resource)
	if err != nil {
		return err
	}
	client.Authorizer = authorizer
	return client.AddToUserAgent(c.userAgent)
}

// AuthorizeClient fetches a resource management authorizer from the environment and injects it to a client.
func (c *Config) AuthorizeClient(client *autorest.Client) (err error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}
	client.Authorizer = authorizer
	return client.AddToUserAgent(c.userAgent)
}

// AuthorizeClientFromFile fetches a resource management authorizer from an SDK auth file and injects it to a client.
func (c *Config) AuthorizeClientFromFile(client *autorest.Client) (err error) {
	authorizer, err := auth.NewAuthorizerFromFile(c.env.ResourceManagerEndpoint)
	if err != nil {
		return err
	}
	client.Authorizer = authorizer
	return client.AddToUserAgent(c.userAgent)
}

// AuthorizeClientFromFileForResource fetches an authorizer from an SDK auth file for an arbitrary resource and injects it to a client.
func (c *Config) AuthorizeClientFromFileForResource(client *autorest.Client, resource string) (err error) {
	authorizer, err := auth.NewAuthorizerFromFileWithResource(resource)
	if err != nil {
		return err
	}
	client.Authorizer = authorizer
	return client.AddToUserAgent(c.userAgent)
}

// GetAuthorizerFromArgs uses arguments passed on the command line to authenticate using AAD app ID, secret, and tenant ID.
// It returns the resultings resource management authorizer.
func (c *Config) GetAuthorizerFromArgs() (autorest.Authorizer, error) {
	return c.GetAuthorizerFromArgsForResource(c.env.ResourceManagerEndpoint)
}

// GetAuthorizerFromArgsForResource uses arguments passed on the command line to authenticate using AAD app ID, secret, and tenant ID.
// It returns an authorizer to the resource provided as an argument.
func (c *Config) GetAuthorizerFromArgsForResource(resource string) (autorest.Authorizer, error) {
	if err := c.validateArgs(); err != nil {
		return nil, err
	}
	authConfig := auth.ClientCredentialsConfig{
		ClientID:     c.app,
		ClientSecret: c.key,
		TenantID:     c.tenant,
		Resource:     resource,
		AADEndpoint:  c.env.ActiveDirectoryEndpoint,
	}

	return authConfig.Authorizer()
}

// AuthorizeClientFromArgs authorizes an SDK client using arguments from the command line.
func (c *Config) AuthorizeClientFromArgs(client *autorest.Client) (err error) {
	return c.AuthorizeClientFromArgsForResource(client, c.env.ResourceManagerEndpoint)
}

// AuthorizeClientFromArgsForResource authorizes an SDK client to the provided resource using arguments from the command line.
func (c *Config) AuthorizeClientFromArgsForResource(client *autorest.Client, resource string) (err error) {
	authorizer, err := c.GetAuthorizerFromArgsForResource(resource)
	if err != nil {
		return err
	}
	client.Authorizer = authorizer
	return client.AddToUserAgent(c.userAgent)
}

func (c *Config) validateArgs() error {
	if c.app == "" || c.tenant == "" || c.key == "" {
		return errors.New("app, tenant, and key must all be provided as options for authenticating with args")
	}
	return nil
}

// GetKeyvaultAuthorizer creates a new Keyvault authorizer.
func (c *Config) GetKeyvaultAuthorizer() (autorest.Authorizer, error) {
	return c.GetAuthorizerFromArgsForResource(strings.TrimSuffix(c.env.KeyVaultEndpoint, "/"))
}
