package authorizer

import (
	"errors"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var _ (Factory) = &CLIAuthorizer{}

type Factory interface {
	New() (autorest.Authorizer, error)
	NewKeyvaultAuthorizer() (autorest.Authorizer, error)
	NewForResource(resource string) (autorest.Authorizer, error)
}

// CLIAuthorizer holds the configured useragent, environment, and authentication
// credentials. The environment will notably be used to identify what resource
// to request tokens for e.g. in sovereign clouds.
type CLIAuthorizer struct {
	userAgent string
	env       azure.Environment
	app       string
	key       string
	tenant    string
}

type CLIOption func(*CLIAuthorizer)

// NewCLIAuthorizer creates an authorizer with
func NewCLIAuthorizer(env string, opts ...CLIOption) (*CLIAuthorizer, error) {
	azureEnv, err := GetEnvironment(env)
	if err != nil {
		return nil, err
	}

	c := &CLIAuthorizer{
		userAgent: "azauth",
		env:       azureEnv,
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := validateCLIArgs(c.app, c.key, c.tenant); err != nil {
		return nil, err
	}

	return c, nil
}

// UserAgent sets the user agent on Azure SDK clients.
func UserAgent(userAgent string) CLIOption {
	return func(c *CLIAuthorizer) {
		c.userAgent = userAgent
	}
}

// App sets the AAD application to authenticate with.
func App(app string) CLIOption {
	return func(c *CLIAuthorizer) {
		c.app = app
	}
}

// Key sets the client secret for the AAD application used in authentication.
func Key(key string) CLIOption {
	return func(c *CLIAuthorizer) {
		c.key = key
	}
}

// Tenant sets the tenant ID for authentication and token acquisition.
func Tenant(tenant string) CLIOption {
	return func(c *CLIAuthorizer) {
		c.tenant = tenant
	}
}

// New uses arguments passed on the command line to authenticate using AAD app ID, secret, and tenant ID.
// It returns the resultings resource management authorizer.
func (c *CLIAuthorizer) New() (autorest.Authorizer, error) {
	return c.NewForResource(c.env.ResourceManagerEndpoint)
}

// NewForResource uses arguments passed on the command line to authenticate using AAD app ID, secret, and tenant ID.
// It returns an authorizer to the resource provided as an argument.
func (c *CLIAuthorizer) NewForResource(resource string) (autorest.Authorizer, error) {
	// TODO(ace): store managment, kv, and other default authorizers (about 6 of them) on the returned object
	creds := auth.ClientCredentialsConfig{
		ClientID:     c.app,
		ClientSecret: c.key,
		TenantID:     c.tenant,
		Resource:     resource,
		AADEndpoint:  c.env.ActiveDirectoryEndpoint,
	}

	return creds.Authorizer()
}

// NewKeyvaultAuthorizer creates a new Keyvault authorizer.
func (c *CLIAuthorizer) NewKeyvaultAuthorizer() (autorest.Authorizer, error) {
	return c.NewForResource(strings.TrimSuffix(c.env.KeyVaultEndpoint, "/"))
}

func validateCLIArgs(app, key, tenant string) error {
	if app == "" || tenant == "" || key == "" {
		return errors.New("app, tenant, and key must all be provided as options for authenticating with args")
	}
	return nil
}

func GetEnvironment(env string) (azure.Environment, error) {
	if env == "" {
		return azure.PublicCloud, nil
	}
	return azure.EnvironmentFromName(env)
}

// // AuthorizeClientForResource fetches an authorizer from the environment for a given resources and injects it to a client.
// func AuthorizeClientForResource(client *autorest.Client, resource string) (err error) {
// 	authorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(resource)
// 	if err != nil {
// 		return err
// 	}
// 	client.Authorizer = authorizer
// 	return client.AddToUserAgent(c.userAgent)
// }

// // AuthorizeClient fetches a resource management authorizer from the environment and injects it to a client.
// func AuthorizeClient(client *autorest.Client) (err error) {
// 	authorizer, err := auth.NewAuthorizerFromEnvironment()
// 	if err != nil {
// 		return err
// 	}
// 	client.Authorizer = authorizer
// 	return client.AddToUserAgent(c.userAgent)
// }

// // AuthorizeClientFromFile fetches a resource management authorizer from an SDK auth file and injects it to a client.
// func AuthorizeClientFromFile(client *autorest.Client) (err error) {
// 	env, err := GetEnvironment()
// 	if err != nil {
// 		return nil, err
// 	}
// 	authorizer, err := auth.NewAuthorizerFromFile(env.ResourceManagerEndpoint)
// 	if err != nil {
// 		return err
// 	}
// 	client.Authorizer = authorizer
// 	return client.AddToUserAgent(c.userAgent)
// }

// // AuthorizeClientFromFileForResource fetches an authorizer from an SDK auth file for an arbitrary resource and injects it to a client.
// func AuthorizeClientFromFileForResource(client *autorest.Client, resource string) (err error) {
// 	authorizer, err := auth.NewAuthorizerFromFileWithResource(resource)
// 	if err != nil {
// 		return err
// 	}
// 	client.Authorizer = authorizer
// 	return client.AddToUserAgent(c.userAgent)
// }
