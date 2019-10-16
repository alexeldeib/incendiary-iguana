package config

import (
	"errors"

	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Config holds environment settings, cached authorizers, and global loggers.
// Notably, the environment settings contain the name of the Azure Cloud,
// required for parameterizing authentication for for each Cloud environment (e.g. Public, Fairfax, Mooncake).
type Config struct {
	userAgent string
	env       *azure.Environment
	app       string
	key       string
	tenant    string
}

type Option func(*Config)

// New fetches and caches environment settings for resource authentication and initializes loggers.
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

// UserAgent provides a method of setting the user agent on the client.
func UserAgent(userAgent string) Option {
	return func(c *Config) {
		c.userAgent = userAgent
	}
}

// App provides a method of setting the user agent on the client.
func App(app string) Option {
	return func(c *Config) {
		c.app = app
	}
}

// Key provides a method of setting the user agent on the client.
func Key(key string) Option {
	return func(c *Config) {
		c.key = key
	}
}

// Tenant provides a method of setting the user agent on the client.
func Tenant(tenant string) Option {
	return func(c *Config) {
		c.tenant = tenant
	}
}

// AuthorizeClientForResource tries to fetch an authorizer using GetAuthorizerForResource and inject it into a client.
func (c *Config) AuthorizeClientForResource(client *autorest.Client, resource string) (err error) {
	if authorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(resource); err == nil {
		client.Authorizer = authorizer
		return client.AddToUserAgent(c.userAgent)
	}
	return
}

// AuthorizeClienet tries to fetch an authorizer for management operations.
func (c *Config) AuthorizeClient(client *autorest.Client) (err error) {
	if authorizer, err := auth.NewAuthorizerFromEnvironment(); err == nil {
		client.Authorizer = authorizer
		return client.AddToUserAgent(c.userAgent)
	}
	return
}

// AuthorizeClientFromFile tries to fetch an authorizer using GetFileAuthorizer and inject it into a client.
func (c *Config) AuthorizeClientFromFile(client *autorest.Client) (err error) {
	if authorizer, err := auth.NewAuthorizerFromFile(c.env.ResourceManagerEndpoint); err == nil {
		client.Authorizer = authorizer
		return client.AddToUserAgent(c.userAgent)
	}
	return
}

// AuthorizeClientFromFile tries to fetch an authorizer using GetFileAuthorizer and inject it into a client.
func (c *Config) AuthorizeClientFromFileForResource(client *autorest.Client, resource string) (err error) {
	if authorizer, err := auth.NewAuthorizerFromFileWithResource(resource); err == nil {
		client.Authorizer = authorizer
		return client.AddToUserAgent(c.userAgent)
	}
	return
}

func (c *Config) GetAuthorizerFromArgs() (autorest.Authorizer, error) {
	if err := c.validateArgs(); err != nil {
		return nil, err
	}
	authConfig := auth.ClientCredentialsConfig{
		ClientID:     c.app,
		ClientSecret: c.key,
		TenantID:     c.tenant,
		Resource:     c.env.ResourceManagerEndpoint,
		AADEndpoint:  c.env.ActiveDirectoryEndpoint,
	}

	return authConfig.Authorizer()
}

// AuthorizeClientFromArgs tries to fetch an authorizer using GetArgsAuthorizer and inject it into a client.
func (c *Config) AuthorizeClientFromArgs(client *autorest.Client) (err error) {
	return c.AuthorizeClientFromArgsForResource(client, c.env.ResourceManagerEndpoint)
}

// AuthorizeClientFromArgs tries to fetch an authorizer using GetArgsAuthorizer and inject it into a client.
func (c *Config) AuthorizeClientFromArgsForResource(client *autorest.Client, resource string) (err error) {
	if authorizer, err := c.GetAuthorizerFromArgs(); err == nil {
		client.Authorizer = authorizer
		return client.AddToUserAgent(c.userAgent)
	}
	return
}

func (c *Config) validateArgs() error {
	if c.app == "" || c.tenant == "" || c.key == "" {
		return errors.New("app, tenant, and key must all be provided as options for authenticating with args")
	}
	return nil
}

// GetKeyvaultAuthorizer creates a new Keyvault authorizer, preferring cli => file => env vars => msi.
func (c *Config) GetKeyvaultAuthorizer() (autorest.Authorizer, error) {
	authorizer, err := kvauth.NewAuthorizerFromFile(azure.PublicCloud.KeyVaultEndpoint)
	if err == nil {
		return authorizer, nil
	}
	authorizer, err = kvauth.NewAuthorizerFromCLI()
	if err == nil {
		return authorizer, nil
	}
	authorizer, err = kvauth.NewAuthorizerFromEnvironment()
	return authorizer, err
}
